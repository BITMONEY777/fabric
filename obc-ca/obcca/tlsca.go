/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package obcca

import (
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"math/big"

	"github.com/golang/protobuf/proto"
	pb "github.com/openblockchain/obc-peer/obc-ca/protos"
	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// TLSCA is the tls certificate authority.
//
type TLSCA struct {
	*CA
	eca     *ECA
}

// TLSCAP serves the public GRPC interface of the TLSCA.
//
type TLSCAP struct {
	tlsca *TLSCA
}

// TLSCAA serves the administrator GRPC interface of the TLS.
//
type TLSCAA struct {
	tlsca *TLSCA
}

// NewTLSCA sets up a new TLSCA.
//
func NewTLSCA(eca *ECA) *TLSCA {
	tlsca := &TLSCA{NewCA("tlsca"), eca}

	return tlsca
}

// Start starts the TLSCA.
//
func (tlsca *TLSCA) Start(srv *grpc.Server) {
	tlsca.startTLSCAP(srv)
	tlsca.startTLSCAA(srv)

	Info.Println("TLSCA started.")
}


func (tlsca *TLSCA) startTLSCAP(srv *grpc.Server) {
	pb.RegisterTLSCAPServer(srv, &TLSCAP{tlsca})
}

func (tlsca *TLSCA) startTLSCAA(srv *grpc.Server) {
	pb.RegisterTLSCAAServer(srv, &TLSCAA{tlsca})
}

// ReadCACertificate reads the certificate of the TLSCA.
//
func (tlscap *TLSCAP) ReadCACertificate(ctx context.Context, in *pb.Empty) (*pb.Cert, error) {
	Trace.Println("grpc TLSCAP:ReadCACertificate")

	return &pb.Cert{tlscap.tlsca.raw}, nil
}

// CreateCertificate requests the creation of a new enrollment certificate by the TLSCA.
//
func (tlscap *TLSCAP) CreateCertificate(ctx context.Context, req *pb.TLSCertCreateReq) (*pb.TLSCertCreateResp, error) {
	Trace.Println("grpc TLSCAP:CreateCertificate")

	id := req.Id.Id

	sig := req.Sig
	req.Sig = nil

	r, s := big.NewInt(0), big.NewInt(0)
	r.UnmarshalText(sig.R)
	s.UnmarshalText(sig.S)

	raw := req.Pub.Key
	if req.Pub.Type != pb.CryptoType_ECDSA {
		return nil, errors.New("unsupported key type")
	}
	pub, err := x509.ParsePKIXPublicKey(req.Pub.Key)
	if err != nil {
		return nil, err
	}

	hash := sha3.New384()
	raw, _ = proto.Marshal(req)
	hash.Write(raw)
	if ecdsa.Verify(pub.(*ecdsa.PublicKey), hash.Sum(nil), r, s) == false {
		return nil, errors.New("signature does not verify")
	}

	if raw, err = tlscap.tlsca.createCertificate(id, pub.(*ecdsa.PublicKey), x509.KeyUsageKeyAgreement, req.Ts.Seconds); err != nil {
		Error.Println(err)
		return nil, err
	}

	return &pb.TLSCertCreateResp{&pb.Cert{raw}}, nil
}

// ReadCertificate reads an enrollment certificate from the TLSCA.
//
func (tlscap *TLSCAP) ReadCertificate(ctx context.Context, req *pb.TLSCertReadReq) (*pb.Cert, error) {
	Trace.Println("grpc TLSCAP:ReadCertificate")

	raw, err := tlscap.tlsca.readCertificate(req.Id.Id, x509.KeyUsageKeyAgreement)
	if err != nil {
		return nil, err
	}

	return &pb.Cert{raw}, nil
}

// RevokeCertificate revokes a certificate from the TLSCA.  Not yet implemented.
//
func (tlscap *TLSCAP) RevokeCertificate(context.Context, *pb.TLSCertRevokeReq) (*pb.CAStatus, error) {
	Trace.Println("grpc TLSCAP:RevokeCertificate")

	return nil, errors.New("not yet implemented")
}

// RevokeCertificate revokes a certificate from the TLSCA.  Not yet implemented.
//
func (tlscaa *TLSCAA) RevokeCertificate(context.Context, *pb.TLSCertRevokeReq) (*pb.CAStatus, error) {
	Trace.Println("grpc TLSCAA:RevokeCertificate")

	return nil, errors.New("not yet implemented")
}