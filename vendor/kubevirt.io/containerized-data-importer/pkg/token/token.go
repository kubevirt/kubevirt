/*
 * This file is part of the CDI project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2019 Red Hat, Inc.
 *
 */

package token

import (
	"crypto/rsa"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// OperationClone is tye of token for cloning a PVC
	OperationClone Operation = "Clone"

	// OperationUpload is the type of token for uploading to a PVC
	OperationUpload Operation = "Upload"
)

// Operation is the type of the token
type Operation string

// Payload is the data inside our token
type Payload struct {
	Operation Operation                   `json:"opertation,omitempty"`
	Name      string                      `json:"name,omitempty"`
	Namespace string                      `json:"namespace,omitempty"`
	Resource  metav1.GroupVersionResource `json:"resource,omitempty"`
	Params    map[string]string           `json:"params,omitempty"`
}

// Validator validates tokens
type Validator interface {
	Validate(string) (*Payload, error)
}

type validator struct {
	issuer string
	key    *rsa.PublicKey
	leeway time.Duration
}

// NewValidator return a new Validator implementation
func NewValidator(issuer string, key *rsa.PublicKey, leeway time.Duration) Validator {
	return &validator{issuer: issuer, key: key, leeway: leeway}
}

// Validate checks the token signature and returns the contents
func (v *validator) Validate(token string) (*Payload, error) {
	tok, err := jwt.ParseSigned(token)
	if err != nil {
		return nil, err
	}

	public := &jwt.Claims{}
	private := &Payload{}

	if err = tok.Claims(v.key, public, private); err != nil {
		return nil, err
	}

	e := jwt.Expected{
		Issuer: v.issuer,
		Time:   time.Now(),
	}

	if err = public.ValidateWithLeeway(e, v.leeway); err != nil {
		return nil, err
	}

	return private, nil
}

// Generator generates tokens
type Generator interface {
	Generate(*Payload) (string, error)
}

type generator struct {
	issuer   string
	key      *rsa.PrivateKey
	lifetime time.Duration
}

// NewGenerator returns a new Generator
func NewGenerator(issuer string, key *rsa.PrivateKey, lifetime time.Duration) Generator {
	return &generator{issuer: issuer, key: key, lifetime: lifetime}
}

// Generate generates a token from the given parameters
func (g *generator) Generate(payload *Payload) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.PS256, Key: g.key}, nil)
	if err != nil {
		return "", errors.Wrap(err, "error creating JWT signer")
	}

	t := time.Now()

	return jwt.Signed(signer).
		Claims(payload).
		Claims(&jwt.Claims{
			Issuer:    g.issuer,
			IssuedAt:  jwt.NewNumericDate(t),
			NotBefore: jwt.NewNumericDate(t),
			Expiry:    jwt.NewNumericDate(t.Add(g.lifetime)),
		}).
		CompactSerialize()
}
