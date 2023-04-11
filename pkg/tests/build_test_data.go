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

package tests

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/digitorus/timestamp"
	"github.com/sigstore/timestamp-authority/pkg/api"
)

func buildJSONReq(t *testing.T, artifact []byte, includeCerts bool, hashAlg string, nonce *big.Int, oidStr string) []byte {
	jsonReq := api.JSONRequest{
		Certificates:  includeCerts,
		HashAlgorithm: hashAlg,
		Artifact:      string(artifact),
		Nonce:         nonce,
		TSAPolicyOID:  oidStr,
	}

	marshalled, err := json.Marshal(jsonReq)
	if err != nil {
		t.Fatalf("failed to marshal request")
	}
	return marshalled
}

func buildTimestampQueryReq(t *testing.T, artifact []byte, opts timestamp.RequestOptions) []byte {
	tsq, err := timestamp.CreateRequest(bytes.NewReader(artifact), &timestamp.RequestOptions{
		Hash:         opts.Hash,
		Certificates: opts.Certificates,
		Nonce:        opts.Nonce,
		TSAPolicyOID: opts.TSAPolicyOID,
	})
	if err != nil {
		t.Fatalf("unexpected error creating request: %v", err)
	}
	return tsq
}
