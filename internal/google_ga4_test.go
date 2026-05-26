package internal

import (
	"context"
	"testing"
)

func TestEnsureGA4DryRunPlansCreates(t *testing.T) {
	result, err := EnsureGA4WebStream(context.Background(), nil, GA4EnsureRequest{
		Account:      "accounts/123",
		PropertyName: "example.com",
		StreamName:   "example.com",
		DefaultURI:   "https://example.com",
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("EnsureGA4WebStream: %v", err)
	}
	if !result.DryRun {
		t.Fatal("dry_run = false")
	}
	if got := operationNames(result.Operations); !sameStrings(got, []string{"create_property", "create_web_data_stream"}) {
		t.Fatalf("operations = %v", got)
	}
}

func TestEnsureGA4ReusesExistingPropertyAndStream(t *testing.T) {
	client := &fakeGA4AdminClient{
		properties: []GA4Property{{Name: "properties/1", DisplayName: "example.com"}},
		streams: map[string][]GA4DataStream{
			"properties/1": {{Name: "properties/1/dataStreams/2", DisplayName: "example.com", DefaultURI: "https://example.com", MeasurementID: "G-ABC123"}},
		},
	}
	result, err := EnsureGA4WebStream(context.Background(), client, GA4EnsureRequest{
		Account:      "accounts/123",
		PropertyName: "example.com",
		StreamName:   "example.com",
		DefaultURI:   "https://example.com",
	})
	if err != nil {
		t.Fatalf("EnsureGA4WebStream: %v", err)
	}
	if result.Property != "properties/1" || result.DataStream != "properties/1/dataStreams/2" || result.MeasurementID != "G-ABC123" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if client.createdProperties != 0 || client.createdStreams != 0 {
		t.Fatalf("created property=%d stream=%d", client.createdProperties, client.createdStreams)
	}
}

func TestEnsureGA4UsesExplicitPropertyWithoutListingOrCreatingProperty(t *testing.T) {
	client := &fakeGA4AdminClient{
		streams: map[string][]GA4DataStream{
			"properties/538139248": {{Name: "properties/538139248/dataStreams/7", DisplayName: "gocodealone.tech", DefaultURI: "https://gocodealone.tech", MeasurementID: "G-VM9JNJRJW1"}},
		},
	}
	result, err := EnsureGA4WebStream(context.Background(), client, GA4EnsureRequest{
		Account:      "accounts/395146029",
		Property:     "properties/538139248",
		PropertyName: "GoCodeAlone",
		StreamName:   "gocodealone.tech",
		DefaultURI:   "https://gocodealone.tech",
	})
	if err != nil {
		t.Fatalf("EnsureGA4WebStream: %v", err)
	}
	if result.Property != "properties/538139248" || result.DataStream != "properties/538139248/dataStreams/7" || result.MeasurementID != "G-VM9JNJRJW1" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if client.listPropertiesCalls != 0 || client.createdProperties != 0 {
		t.Fatalf("listed properties=%d created properties=%d", client.listPropertiesCalls, client.createdProperties)
	}
}

func TestEnsureGA4ExplicitPropertyRequiresClientForLiveEnsure(t *testing.T) {
	_, err := EnsureGA4WebStream(context.Background(), nil, GA4EnsureRequest{
		Account:    "accounts/395146029",
		Property:   "properties/538139248",
		StreamName: "gocodealone.tech",
		DefaultURI: "https://gocodealone.tech",
	})
	if err == nil {
		t.Fatal("expected credentials error")
	}
}

func TestEnsureGA4CreatesMissingResources(t *testing.T) {
	client := &fakeGA4AdminClient{}
	result, err := EnsureGA4WebStream(context.Background(), client, GA4EnsureRequest{
		Account:      "accounts/123",
		PropertyName: "example.com",
		StreamName:   "example.com",
		DefaultURI:   "https://example.com",
	})
	if err != nil {
		t.Fatalf("EnsureGA4WebStream: %v", err)
	}
	if result.Property == "" || result.DataStream == "" || result.MeasurementID == "" {
		t.Fatalf("missing created IDs: %#v", result)
	}
	if got := operationNames(result.Operations); !sameStrings(got, []string{"create_property", "create_web_data_stream"}) {
		t.Fatalf("operations = %v", got)
	}
}

func TestEnsureGA4RejectsInvalidInput(t *testing.T) {
	_, err := EnsureGA4WebStream(context.Background(), nil, GA4EnsureRequest{
		Account:      "123",
		PropertyName: "example.com",
		StreamName:   "example.com",
		DefaultURI:   "ftp://example.com",
		DryRun:       true,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

type fakeGA4AdminClient struct {
	properties          []GA4Property
	streams             map[string][]GA4DataStream
	listPropertiesCalls int
	createdProperties   int
	createdStreams      int
}

func (f *fakeGA4AdminClient) ListProperties(_ context.Context, account string) ([]GA4Property, error) {
	f.listPropertiesCalls++
	return f.properties, nil
}

func (f *fakeGA4AdminClient) CreateProperty(_ context.Context, req GA4CreatePropertyRequest) (GA4Property, error) {
	f.createdProperties++
	p := GA4Property{Name: "properties/created", DisplayName: req.DisplayName}
	f.properties = append(f.properties, p)
	return p, nil
}

func (f *fakeGA4AdminClient) ListDataStreams(_ context.Context, property string) ([]GA4DataStream, error) {
	if f.streams == nil {
		return nil, nil
	}
	return f.streams[property], nil
}

func (f *fakeGA4AdminClient) CreateWebDataStream(_ context.Context, req GA4CreateWebDataStreamRequest) (GA4DataStream, error) {
	f.createdStreams++
	s := GA4DataStream{Name: req.Property + "/dataStreams/created", DisplayName: req.DisplayName, DefaultURI: req.DefaultURI, MeasurementID: "G-CREATED"}
	if f.streams == nil {
		f.streams = make(map[string][]GA4DataStream)
	}
	f.streams[req.Property] = append(f.streams[req.Property], s)
	return s, nil
}
