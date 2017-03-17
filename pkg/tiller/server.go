/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tiller

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/version"

	authenticationapi "k8s.io/kubernetes/pkg/apis/authentication"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	rest "k8s.io/kubernetes/pkg/client/restclient"
)

// maxMsgSize use 10MB as the default message size limit.
// grpc library default is 4MB
var maxMsgSize = 1024 * 1024 * 10

// NewServer creates a new grpc server.
func NewServer(syscfg *rest.Config) *grpc.Server {
	return grpc.NewServer(
		grpc.MaxMsgSize(maxMsgSize),
		grpc.UnaryInterceptor(newUnaryInterceptor(syscfg)),
		grpc.StreamInterceptor(newStreamInterceptor(syscfg)),
	)
}

func authenticate(ctx context.Context, syscfg *rest.Config) (context.Context, error) {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return nil, errors.New("Missing metadata in context.")
	}

	var user *authenticationapi.UserInfo
	var usrcfg *rest.Config
	var err error
	authHeader, ok := md[string(kube.Authorization)]
	if !ok || len(authHeader) == 0 || authHeader[0] == "" {
		user, usrcfg, err = checkClientCert(ctx, syscfg)
	} else {
		if strings.HasPrefix(authHeader[0], "Bearer ") {
			user, usrcfg, err = checkBearerAuth(authHeader[0], syscfg)
		} else if strings.HasPrefix(authHeader[0], "Basic ") {
			user, usrcfg, err = checkBasicAuth(authHeader[0], syscfg)
		} else {
			return nil, errors.New("Unknown authorization scheme.")
		}
	}
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, kube.UserInfo, user)
	ctx = context.WithValue(ctx, kube.UserClientConfig, usrcfg)
	ctx = context.WithValue(ctx, kube.SystemClientConfig, syscfg)

	return ctx, nil
}

func newUnaryInterceptor(syscfg *rest.Config) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		err = checkClientVersion(ctx)
		if err != nil {
			// whitelist GetVersion() from the version check
			if _, m := splitMethod(info.FullMethod); m != "GetVersion" {
				log.Println(err)
				return nil, err
			}
		}
		ctx, err = authenticate(ctx, syscfg)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return handler(ctx, req)
	}
}

func newStreamInterceptor(syscfg *rest.Config) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		err := checkClientVersion(ctx)
		if err != nil {
			log.Println(err)
			return err
		}
		ctx, err = authenticate(ctx, syscfg)
		if err != nil {
			log.Println(err)
			return err
		}

		newStream := serverStreamWrapper{
			ss:  ss,
			ctx: ctx,
		}
		return handler(srv, newStream)
	}
}

// serverStreamWrapper wraps original ServerStream but uses modified context.
// this modified context will be available inside handler()
type serverStreamWrapper struct {
	ss  grpc.ServerStream
	ctx context.Context
}

func (w serverStreamWrapper) Context() context.Context        { return w.ctx }
func (w serverStreamWrapper) RecvMsg(msg interface{}) error   { return w.ss.RecvMsg(msg) }
func (w serverStreamWrapper) SendMsg(msg interface{}) error   { return w.ss.SendMsg(msg) }
func (w serverStreamWrapper) SendHeader(md metadata.MD) error { return w.ss.SendHeader(md) }
func (w serverStreamWrapper) SetHeader(md metadata.MD) error  { return w.ss.SetHeader(md) }
func (w serverStreamWrapper) SetTrailer(md metadata.MD)       { w.ss.SetTrailer(md) }

func splitMethod(fullMethod string) (string, string) {
	if frags := strings.Split(fullMethod, "/"); len(frags) == 3 {
		return frags[1], frags[2]
	}
	return "unknown", "unknown"
}

func versionFromContext(ctx context.Context) string {
	if md, ok := metadata.FromContext(ctx); ok {
		if v, ok := md["x-helm-api-client"]; ok && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func checkClientVersion(ctx context.Context) error {
	clientVersion := versionFromContext(ctx)
	if !version.IsCompatible(clientVersion, version.Version) {
		return fmt.Errorf("incompatible versions client: %s server: %s", clientVersion, version.Version)
	}
	return nil
}

func checkBearerAuth(h string, syscfg *rest.Config) (*authenticationapi.UserInfo, *rest.Config, error) {
	token := h[len("Bearer "):]

	client, err := clientset.NewForConfig(syscfg)
	if err != nil {
		return nil, nil, err
	}

	// verify token
	tokenReq := &authenticationapi.TokenReview{
		Spec: authenticationapi.TokenReviewSpec{
			Token: token,
		},
	}
	result, err := client.AuthenticationClient.TokenReviews().Create(tokenReq)
	if err != nil {
		return nil, nil, err
	}
	if !result.Status.Authenticated {
		return nil, nil, errors.New("Not authenticated")
	}

	usrcfg := &rest.Config{
		Host:        syscfg.Host,
		APIPath:     syscfg.APIPath,
		Prefix:      syscfg.Prefix,
		BearerToken: token,
	}
	usrcfg.TLSClientConfig.CertData = syscfg.TLSClientConfig.CertData
	return &result.Status.User, usrcfg, nil
}

func checkBasicAuth(h string, syscfg *rest.Config) (*authenticationapi.UserInfo, *rest.Config, error) {
	basicAuth, err := base64.StdEncoding.DecodeString(h[len("Basic "):])
	if err != nil {
		return nil, nil, err
	}
	username, password := getUserPasswordFromBasicAuth(string(basicAuth))
	if len(username) == 0 || len(password) == 0 {
		return nil, nil, errors.New("Missing username or password.")
	}

	usrcfg := &rest.Config{
		Host:     syscfg.Host,
		APIPath:  syscfg.APIPath,
		Prefix:   syscfg.Prefix,
		Username: username,
		Password: password,
	}
	usrcfg.TLSClientConfig.CertData = syscfg.TLSClientConfig.CertData

	client, err := clientset.NewForConfig(usrcfg)
	if err != nil {
		return nil, nil, err
	}

	// verify credentials
	_, err = client.DiscoveryClient.ServerVersion()
	if err != nil {
		return nil, nil, err
	}

	return &authenticationapi.UserInfo{
		Username: username,
	}, usrcfg, nil
}

func getUserPasswordFromBasicAuth(token string) (string, string) {
	st := strings.SplitN(token, ":", 2)
	if len(st) == 2 {
		return st[0], st[1]
	}
	return "", ""
}

func checkClientCert(ctx context.Context, syscfg *rest.Config) (*authenticationapi.UserInfo, *rest.Config, error) {
	// ref: https://github.com/grpc/grpc-go/issues/111#issuecomment-275820771
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return nil, nil, errors.New("No peer found!")
	}
	tlsInfo, ok := peer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, nil, errors.New("No TLS credential found!")
	}
	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return nil, nil, errors.New("No verified client certificate found!")
	}

	c := tlsInfo.State.VerifiedChains[0][0]
	user := authenticationapi.UserInfo{
		Username: c.Subject.CommonName,
	}
	usrcfg := *syscfg
	usrcfg.Impersonate = c.Subject.CommonName
	return &user, &usrcfg, nil
}
