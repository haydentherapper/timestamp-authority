// Copyright 2022 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package signer

import (
	"context"
	"crypto"
	"crypto/elliptic"
	"crypto/rand"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/kms"
)

const MemoryScheme = "memory"

func NewCryptoSigner(ctx context.Context, signer string) (crypto.Signer, error) {
	switch {
	case signer == MemoryScheme:
		sv, _, err := signature.NewECDSASignerVerifier(elliptic.P256(), rand.Reader, crypto.SHA256)
		return sv, err
	default:
		signer, err := kms.Get(ctx, signer, crypto.SHA256)
		if err != nil {
			return nil, err
		}
		s, _, err := signer.CryptoSigner(ctx, func(err error) {})
		return s, err
	}
}
