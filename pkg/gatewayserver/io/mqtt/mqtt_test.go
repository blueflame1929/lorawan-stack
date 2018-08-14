// Copyright © 2018 The Things Network Foundation, The Things Industries B.V.
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

package mqtt_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogo/protobuf/proto"
	"github.com/smartystreets/assertions"
	"go.thethings.network/lorawan-stack/pkg/gatewayserver/io"
	. "go.thethings.network/lorawan-stack/pkg/gatewayserver/io/mqtt"
	iotesting "go.thethings.network/lorawan-stack/pkg/gatewayserver/io/testing"
	"go.thethings.network/lorawan-stack/pkg/log"
	"go.thethings.network/lorawan-stack/pkg/ttnpb"
	"go.thethings.network/lorawan-stack/pkg/util/test"
	"go.thethings.network/lorawan-stack/pkg/util/test/assertions/should"
)

var (
	registeredGatewayUID = "test-gateway"
	registeredGatewayID  = ttnpb.GatewayIdentifiers{GatewayID: "test-gateway"}
	registeredGatewayKey = "test-key"

	timeout = 10 * time.Millisecond
)

func TestAuthentication(t *testing.T) {
	a := assertions.New(t)

	ctx := log.NewContext(test.Context(), test.GetLogger(t))
	ctx = newContextWithRightsFetcher(ctx)
	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	gs := iotesting.NewServer()
	lis, err := net.Listen("tcp", ":0")
	if !a.So(err, should.BeNil) {
		t.FailNow()
	}
	Start(ctx, gs, lis, "tcp")

	for _, tc := range []struct {
		UID string
		Key string
		OK  bool
	}{
		{
			UID: registeredGatewayUID,
			Key: registeredGatewayKey,
			OK:  true,
		},
		{
			UID: registeredGatewayUID,
			Key: "invalid-key",
			OK:  false,
		},
		{
			UID: "invalid-gateway",
			Key: "invalid-key",
			OK:  false,
		},
	} {
		t.Run(fmt.Sprintf("%v:%v", tc.UID, tc.Key), func(t *testing.T) {
			a := assertions.New(t)

			clientOpts := mqtt.NewClientOptions()
			clientOpts.AddBroker(fmt.Sprintf("tcp://%v", lis.Addr()))
			clientOpts.SetUsername(tc.UID)
			clientOpts.SetPassword(tc.Key)
			client := mqtt.NewClient(clientOpts)
			token := client.Connect()
			if ok := token.WaitTimeout(timeout); tc.OK {
				if a.So(ok, should.BeTrue) && a.So(token.Error(), should.BeNil) {
					client.Disconnect(100)
				}
			} else {
				a.So(ok, should.BeFalse)
			}
		})
	}
}

func TestTraffic(t *testing.T) {
	a := assertions.New(t)

	ctx := log.NewContext(test.Context(), test.GetLogger(t))
	ctx = newContextWithRightsFetcher(ctx)
	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	gs := iotesting.NewServer()
	lis, err := net.Listen("tcp", ":0")
	if !a.So(err, should.BeNil) {
		t.FailNow()
	}
	Start(ctx, gs, lis, "tcp")

	clientOpts := mqtt.NewClientOptions()
	clientOpts.AddBroker(fmt.Sprintf("tcp://%v", lis.Addr()))
	clientOpts.SetUsername(registeredGatewayUID)
	clientOpts.SetPassword(registeredGatewayKey)
	client := mqtt.NewClient(clientOpts)
	client.Connect()

	var conn *io.Connection
	select {
	case conn = <-gs.Connections():
	case <-time.After(timeout):
		t.Fatal("Connection timeout")
	}
	defer client.Disconnect(100)

	t.Run("Upstream", func(t *testing.T) {
		for _, tc := range []struct {
			Topic   string
			Message proto.Marshaler
			OK      bool
		}{
			{
				Topic: fmt.Sprintf("v3/%v/up", registeredGatewayID.GatewayID),
				Message: &ttnpb.UplinkMessage{
					RawPayload: []byte{0x01},
				},
				OK: true,
			},
			{
				Topic: fmt.Sprintf("v3/%v/up", registeredGatewayID.GatewayID),
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x02},
				},
				OK: false, // downlink to an uplink topic
			},
			{
				Topic: fmt.Sprintf("v3/%v/up", "invalid-gateway"),
				Message: &ttnpb.UplinkMessage{
					RawPayload: []byte{0x03},
				},
				OK: false, // invalid gateway ID
			},
			{
				Topic: fmt.Sprintf("v3/%v/down", registeredGatewayID.GatewayID),
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x04},
				},
				OK: false, // publish to downlink not permitted
			},
			{
				Topic: fmt.Sprintf("v3/%v/down", "invalid-gateway"),
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x05},
				},
				OK: false, // invalid gateway ID + publish to downlink not permitted
			},
			{
				Topic: fmt.Sprintf("v3/%v/status", registeredGatewayID.GatewayID),
				Message: &ttnpb.GatewayStatus{
					IP: []string{"1.1.1.1"},
				},
				OK: true,
			},
			{
				Topic: fmt.Sprintf("v3/%v/status", "invalid-gateway"),
				Message: &ttnpb.GatewayStatus{
					IP: []string{"2.2.2.2"},
				},
				OK: false, // invalid gateway ID
			},
			{
				Topic: "invalid/format",
				Message: &ttnpb.GatewayStatus{
					IP: []string{"3.3.3.3"},
				},
				OK: false, // invalid topic format
			},
		} {
			t.Run(tc.Topic, func(t *testing.T) {
				a := assertions.New(t)
				buf, err := tc.Message.Marshal()
				a.So(err, should.BeNil)
				if token := client.Publish(tc.Topic, 1, false, buf); !a.So(token.WaitTimeout(timeout), should.BeTrue) {
					t.FailNow()
				}
				if tc.OK {
					select {
					case up := <-conn.Up():
						a.So(up, should.Resemble, tc.Message)
					case status := <-conn.Status():
						a.So(status, should.Resemble, tc.Message)
					case <-time.After(timeout):
						t.Fatal("Receive expected upstream timeout")
					}
				}
			})
		}
	})

	t.Run("Downstream", func(t *testing.T) {
		for _, tc := range []struct {
			Topic   string
			Message *ttnpb.DownlinkMessage
			OK      bool
		}{
			{
				Topic: fmt.Sprintf("v3/%v/down", registeredGatewayID.GatewayID),
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x01},
					Settings: ttnpb.TxSettings{
						Modulation:      ttnpb.Modulation_LORA,
						Bandwidth:       125000,
						SpreadingFactor: 7,
						CodingRate:      "4/5",
						Frequency:       869525000,
					},
				},
				OK: true,
			},
			{
				Topic: fmt.Sprintf("v3/%v/down", registeredGatewayID.GatewayID),
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x02},
				},
				OK: false, // Tx settings missing
			},
			{
				Topic: fmt.Sprintf("v3/%v/down", "invalid-gateway"),
				Message: &ttnpb.DownlinkMessage{
					RawPayload: []byte{0x03},
					Settings: ttnpb.TxSettings{
						Modulation:      ttnpb.Modulation_LORA,
						Bandwidth:       125000,
						SpreadingFactor: 7,
						CodingRate:      "4/5",
						Frequency:       869525000,
					},
				},
				OK: false, // invalid gateway ID
			},
		} {
			t.Run(tc.Topic, func(t *testing.T) {
				a := assertions.New(t)

				downCh := make(chan *ttnpb.DownlinkMessage)
				handler := func(_ mqtt.Client, msg mqtt.Message) {
					down := &ttnpb.GatewayDown{}
					err := down.Unmarshal(msg.Payload())
					a.So(err, should.BeNil)
					downCh <- down.DownlinkMessage
				}
				if token := client.Subscribe(tc.Topic, 1, handler); !a.So(token.WaitTimeout(timeout), should.BeTrue) {
					t.FailNow()
				}

				err := conn.SendDown(tc.Message)
				if tc.OK {
					if !a.So(err, should.BeNil) {
						t.FailNow()
					}
					select {
					case down := <-downCh:
						a.So(down, should.Resemble, tc.Message)
					case <-time.After(timeout):
						t.Fatal("Receive expected downlink timeout")
					}
				} else if !a.So(err, should.NotBeNil) {
					t.FailNow()
				}
			})
		}
	})
}