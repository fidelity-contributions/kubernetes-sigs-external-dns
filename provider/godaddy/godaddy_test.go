/*
Copyright 2017 The Kubernetes Authors.

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

package godaddy

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

type mockGoDaddyClient struct {
	mock.Mock
	currentTest *testing.T
}

func newMockGoDaddyClient(t *testing.T) *mockGoDaddyClient {
	return &mockGoDaddyClient{
		currentTest: t,
	}
}

var (
	zoneNameExampleOrg string = "example.org"
	zoneNameExampleNet string = "example.net"
)

func (c *mockGoDaddyClient) Post(endpoint string, input interface{}, output interface{}) error {
	log.Infof("POST: %s - %v", endpoint, input)
	stub := c.Called(endpoint, input)
	data, err := json.Marshal(stub.Get(0))
	require.NoError(c.currentTest, err)
	err = json.Unmarshal(data, output)
	require.NoError(c.currentTest, err)
	return stub.Error(1)
}

func (c *mockGoDaddyClient) Patch(endpoint string, input interface{}, output interface{}) error {
	log.Infof("PATCH: %s - %v", endpoint, input)
	stub := c.Called(endpoint, input)
	data, err := json.Marshal(stub.Get(0))
	require.NoError(c.currentTest, err)
	err = json.Unmarshal(data, output)
	require.NoError(c.currentTest, err)
	return stub.Error(1)
}

func (c *mockGoDaddyClient) Put(endpoint string, input interface{}, output interface{}) error {
	log.Infof("PUT: %s - %v", endpoint, input)
	stub := c.Called(endpoint, input)
	data, err := json.Marshal(stub.Get(0))
	require.NoError(c.currentTest, err)
	err = json.Unmarshal(data, output)
	require.NoError(c.currentTest, err)
	return stub.Error(1)
}

func (c *mockGoDaddyClient) Get(endpoint string, output interface{}) error {
	log.Infof("GET: %s", endpoint)
	stub := c.Called(endpoint)
	data, err := json.Marshal(stub.Get(0))
	require.NoError(c.currentTest, err)
	err = json.Unmarshal(data, output)
	require.NoError(c.currentTest, err)
	return stub.Error(1)
}

func (c *mockGoDaddyClient) Delete(endpoint string, output interface{}) error {
	log.Infof("DELETE: %s", endpoint)
	stub := c.Called(endpoint)
	data, err := json.Marshal(stub.Get(0))
	require.NoError(c.currentTest, err)
	err = json.Unmarshal(data, output)
	require.NoError(c.currentTest, err)
	return stub.Error(1)
}

func TestGoDaddyZones(t *testing.T) {
	assert := assert.New(t)
	client := newMockGoDaddyClient(t)
	provider := &GDProvider{
		client:       client,
		domainFilter: endpoint.NewDomainFilter([]string{"com"}),
	}

	// Basic zones
	client.On("Get", domainsURI).Return([]gdZone{
		{
			Domain: "example.com",
		},
		{
			Domain: "example.net",
		},
	}, nil).Once()

	domains, err := provider.zones()

	assert.NoError(err)
	assert.Contains(domains, "example.com")
	assert.NotContains(domains, "example.net")

	client.AssertExpectations(t)

	// Error on getting zones
	client.On("Get", domainsURI).Return(nil, ErrAPIDown).Once()
	domains, err = provider.zones()
	assert.Error(err)
	assert.Nil(domains)
	client.AssertExpectations(t)
}

func TestGoDaddyZoneRecords(t *testing.T) {
	assert := assert.New(t)
	client := newMockGoDaddyClient(t)
	provider := &GDProvider{
		client: client,
	}

	// Basic zones records
	client.On("Get", domainsURI).Return([]gdZone{
		{
			Domain: zoneNameExampleNet,
		},
	}, nil).Once()

	client.On("Get", "/v1/domains/example.net/records").Return([]gdRecordField{
		{
			Name: "godaddy",
			Type: "NS",
			TTL:  defaultTTL,
			Data: "203.0.113.42",
		},
		{
			Name: "godaddy",
			Type: "A",
			TTL:  defaultTTL,
			Data: "203.0.113.42",
		},
	}, nil).Once()

	zones, records, err := provider.zonesRecords(context.TODO(), true)

	assert.NoError(err)

	assert.ElementsMatch(zones, []string{
		zoneNameExampleNet,
	})

	assert.ElementsMatch(records, []gdRecords{
		{
			zone: zoneNameExampleNet,
			records: []gdRecordField{
				{
					Name: "godaddy",
					Type: "NS",
					TTL:  defaultTTL,
					Data: "203.0.113.42",
				},
				{
					Name: "godaddy",
					Type: "A",
					TTL:  defaultTTL,
					Data: "203.0.113.42",
				},
			},
		},
	})

	client.AssertExpectations(t)

	// Error on getting zones list
	client.On("Get", domainsURI).Return(nil, ErrAPIDown).Once()
	zones, records, err = provider.zonesRecords(context.TODO(), false)
	assert.Error(err)
	assert.Nil(zones)
	assert.Nil(records)
	client.AssertExpectations(t)

	// Error on getting zone records
	client.On("Get", domainsURI).Return([]gdZone{
		{
			Domain: zoneNameExampleNet,
		},
	}, nil).Once()

	client.On("Get", "/v1/domains/example.net/records").Return(nil, ErrAPIDown).Once()

	zones, records, err = provider.zonesRecords(context.TODO(), false)

	assert.Error(err)
	assert.Nil(zones)
	assert.Nil(records)
	client.AssertExpectations(t)

	// Error on getting zone record detail
	client.On("Get", domainsURI).Return([]gdZone{
		{
			Domain: zoneNameExampleNet,
		},
	}, nil).Once()

	client.On("Get", "/v1/domains/example.net/records").Return(nil, ErrAPIDown).Once()

	zones, records, err = provider.zonesRecords(context.TODO(), false)
	assert.Error(err)
	assert.Nil(zones)
	assert.Nil(records)
	client.AssertExpectations(t)
}

func TestGoDaddyRecords(t *testing.T) {
	assert := assert.New(t)
	client := newMockGoDaddyClient(t)
	provider := &GDProvider{
		client: client,
	}

	// Basic zones records
	client.On("Get", domainsURI).Return([]gdZone{
		{
			Domain: zoneNameExampleOrg,
		},
		{
			Domain: zoneNameExampleNet,
		},
	}, nil).Once()

	client.On("Get", "/v1/domains/example.org/records").Return([]gdRecordField{
		{
			Name: "@",
			Type: "A",
			TTL:  defaultTTL,
			Data: "203.0.113.42",
		},
		{
			Name: "www",
			Type: "CNAME",
			TTL:  defaultTTL,
			Data: "example.org",
		},
	}, nil).Once()

	client.On("Get", "/v1/domains/example.net/records").Return([]gdRecordField{
		{
			Name: "godaddy",
			Type: "A",
			TTL:  defaultTTL,
			Data: "203.0.113.42",
		},
		{
			Name: "godaddy",
			Type: "A",
			TTL:  defaultTTL,
			Data: "203.0.113.43",
		},
	}, nil).Once()

	endpoints, err := provider.Records(context.TODO())
	assert.NoError(err)

	// Little fix for multi targets endpoint
	for _, endpoint := range endpoints {
		sort.Strings(endpoint.Targets)
	}

	assert.ElementsMatch(endpoints, []*endpoint.Endpoint{
		{
			DNSName:    "godaddy.example.net",
			RecordType: "A",
			RecordTTL:  defaultTTL,
			Labels:     endpoint.NewLabels(),
			Targets: []string{
				"203.0.113.42",
				"203.0.113.43",
			},
		},
		{
			DNSName:    "example.org",
			RecordType: "A",
			RecordTTL:  defaultTTL,
			Labels:     endpoint.NewLabels(),
			Targets: []string{
				"203.0.113.42",
			},
		},
		{
			DNSName:    "www.example.org",
			RecordType: "CNAME",
			RecordTTL:  defaultTTL,
			Labels:     endpoint.NewLabels(),
			Targets: []string{
				"example.org",
			},
		},
	})

	client.AssertExpectations(t)

	// Error getting zone
	client.On("Get", domainsURI).Return(nil, ErrAPIDown).Once()
	endpoints, err = provider.Records(context.TODO())
	assert.Error(err)
	assert.Nil(endpoints)
	client.AssertExpectations(t)
}

func TestGoDaddyChange(t *testing.T) {
	assert := assert.New(t)
	client := newMockGoDaddyClient(t)
	provider := &GDProvider{
		client: client,
	}

	changes := plan.Changes{
		Create: []*endpoint.Endpoint{
			{
				DNSName:    ".example.net",
				RecordType: "A",
				RecordTTL:  defaultTTL,
				Targets: []string{
					"203.0.113.42",
				},
			},
		},
		Delete: []*endpoint.Endpoint{
			{
				DNSName:    "godaddy.example.net",
				RecordType: "A",
				Targets: []string{
					"203.0.113.43",
				},
			},
		},
	}

	// Fetch domains
	client.On("Get", domainsURI).Return([]gdZone{
		{
			Domain: zoneNameExampleNet,
		},
	}, nil).Once()

	// Fetch record
	client.On("Get", "/v1/domains/example.net/records").Return([]gdRecordField{
		{
			Name: "godaddy",
			Type: "A",
			TTL:  defaultTTL,
			Data: "203.0.113.43",
		},
	}, nil).Once()

	// Add entry
	client.On("Patch", "/v1/domains/example.net/records", []gdRecordField{
		{
			Name: "@",
			Type: "A",
			TTL:  defaultTTL,
			Data: "203.0.113.42",
		},
	}).Return(nil, nil).Once()

	// Delete entry
	client.On("Delete", "/v1/domains/example.net/records/A/godaddy").Return(nil, nil).Once()

	assert.NoError(provider.ApplyChanges(context.TODO(), &changes))

	client.AssertExpectations(t)
}

const (
	operationFailedTestErrCode = "GD500"
	operationFailedTestReason  = "Could not apply request"
	recordNotFoundErrCode      = "GD404"
	recordNotFoundReason       = "The requested record is not found in DNS zone"
)

func TestGoDaddyErrorResponse(t *testing.T) {
	assert := assert.New(t)
	client := newMockGoDaddyClient(t)
	provider := &GDProvider{
		client: client,
	}

	changes := plan.Changes{
		Create: []*endpoint.Endpoint{
			{
				DNSName:    ".example.net",
				RecordType: "A",
				RecordTTL:  defaultTTL,
				Targets: []string{
					"203.0.113.42",
				},
			},
		},
		Delete: []*endpoint.Endpoint{
			{
				DNSName:    "godaddy.example.net",
				RecordType: "A",
				Targets: []string{
					"203.0.113.43",
				},
			},
		},
	}

	// Fetch domains
	client.On("Get", domainsURI).Return([]gdZone{
		{
			Domain: zoneNameExampleNet,
		},
	}, nil).Once()

	// Fetch record
	client.On("Get", "/v1/domains/example.net/records").Return([]gdRecordField{
		{
			Name: "godaddy",
			Type: "A",
			TTL:  defaultTTL,
			Data: "203.0.113.43",
		},
	}, nil).Once()

	// Delete entry
	client.On("Delete", "/v1/domains/example.net/records/A/godaddy").Return(GDErrorResponse{
		Code:    operationFailedTestErrCode,
		Message: operationFailedTestReason,
		Fields: []GDErrorField{{
			Code:    recordNotFoundErrCode,
			Message: recordNotFoundReason,
		}},
	}, errors.New(operationFailedTestReason)).Once()

	assert.Error(provider.ApplyChanges(context.TODO(), &changes))

	client.AssertExpectations(t)
}
