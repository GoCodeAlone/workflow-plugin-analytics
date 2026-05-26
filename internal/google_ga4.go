package internal

import (
	"context"
	"fmt"
	"strings"

	admin "cloud.google.com/go/analytics/admin/apiv1alpha"
	"cloud.google.com/go/analytics/admin/apiv1alpha/adminpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GA4EnsureRequest struct {
	Account      string `json:"account"`
	Property     string `json:"property,omitempty"`
	PropertyName string `json:"property_name"`
	StreamName   string `json:"stream_name"`
	DefaultURI   string `json:"default_uri"`
	TimeZone     string `json:"time_zone,omitempty"`
	CurrencyCode string `json:"currency_code,omitempty"`
	DryRun       bool   `json:"dry_run"`
}

type GA4EnsureResult struct {
	Account       string      `json:"account"`
	Property      string      `json:"property"`
	DataStream    string      `json:"data_stream"`
	MeasurementID string      `json:"measurement_id"`
	DryRun        bool        `json:"dry_run"`
	Operations    []Operation `json:"operations"`
}

type GA4Property struct {
	Name        string
	DisplayName string
}

type GA4DataStream struct {
	Name          string
	DisplayName   string
	DefaultURI    string
	MeasurementID string
}

type GA4CreatePropertyRequest struct {
	Account     string
	DisplayName string
	TimeZone    string
	Currency    string
}

type GA4CreateWebDataStreamRequest struct {
	Property    string
	DisplayName string
	DefaultURI  string
}

type GA4AdminClient interface {
	ListProperties(ctx context.Context, account string) ([]GA4Property, error)
	CreateProperty(ctx context.Context, req GA4CreatePropertyRequest) (GA4Property, error)
	ListDataStreams(ctx context.Context, property string) ([]GA4DataStream, error)
	CreateWebDataStream(ctx context.Context, req GA4CreateWebDataStreamRequest) (GA4DataStream, error)
}

func EnsureGA4WebStream(ctx context.Context, client GA4AdminClient, req GA4EnsureRequest) (GA4EnsureResult, error) {
	req.Property = strings.TrimSpace(req.Property)
	req.PropertyName = strings.TrimSpace(req.PropertyName)
	req.StreamName = strings.TrimSpace(req.StreamName)
	req.DefaultURI = strings.TrimSpace(req.DefaultURI)
	if err := validateGoogleAccount(req.Account); err != nil {
		return GA4EnsureResult{}, err
	}
	if req.Property != "" {
		if err := validateGoogleProperty(req.Property); err != nil {
			return GA4EnsureResult{}, err
		}
	}
	if req.PropertyName == "" && req.Property == "" {
		return GA4EnsureResult{}, fmt.Errorf("property_name is required")
	}
	if req.StreamName == "" {
		req.StreamName = defaultString(req.PropertyName, req.Property)
	}
	if err := validateWebURI(req.DefaultURI); err != nil {
		return GA4EnsureResult{}, err
	}
	result := GA4EnsureResult{Account: req.Account, DryRun: req.DryRun}

	var property GA4Property
	if req.Property != "" {
		property = GA4Property{Name: req.Property, DisplayName: req.PropertyName}
		result.Operations = append(result.Operations, reused("reuse_property"))
	} else if !req.DryRun {
		if client == nil {
			return GA4EnsureResult{}, fmt.Errorf("google credentials are required for live GA4 ensure")
		}
		properties, err := client.ListProperties(ctx, req.Account)
		if err != nil {
			return result, err
		}
		for _, candidate := range properties {
			if candidate.DisplayName == req.PropertyName {
				property = candidate
				result.Operations = append(result.Operations, reused("reuse_property"))
				break
			}
		}
	}
	if property.Name == "" {
		result.Operations = append(result.Operations, planned("create_property", req.DryRun))
		if req.DryRun {
			result.Property = ""
			result.Operations = append(result.Operations, planned("create_web_data_stream", true))
			return result, nil
		}
		created, err := client.CreateProperty(ctx, GA4CreatePropertyRequest{
			Account:     req.Account,
			DisplayName: req.PropertyName,
			TimeZone:    defaultString(req.TimeZone, "America/New_York"),
			Currency:    defaultString(req.CurrencyCode, "USD"),
		})
		if err != nil {
			return result, err
		}
		property = created
	}
	result.Property = property.Name
	if req.DryRun {
		result.Operations = append(result.Operations, planned("create_web_data_stream", true))
		return result, nil
	}
	if client == nil {
		return GA4EnsureResult{}, fmt.Errorf("google credentials are required for live GA4 ensure")
	}

	var stream GA4DataStream
	streams, err := client.ListDataStreams(ctx, property.Name)
	if err != nil {
		return result, err
	}
	for _, candidate := range streams {
		if candidate.DisplayName == req.StreamName || candidate.DefaultURI == req.DefaultURI {
			stream = candidate
			result.Operations = append(result.Operations, reused("reuse_web_data_stream"))
			break
		}
	}
	if stream.Name == "" {
		result.Operations = append(result.Operations, planned("create_web_data_stream", false))
		created, err := client.CreateWebDataStream(ctx, GA4CreateWebDataStreamRequest{
			Property:    property.Name,
			DisplayName: req.StreamName,
			DefaultURI:  req.DefaultURI,
		})
		if err != nil {
			return result, err
		}
		stream = created
	}
	result.DataStream = stream.Name
	result.MeasurementID = stream.MeasurementID
	return result, nil
}

type googleGA4SDKClient struct {
	client *admin.AnalyticsAdminClient
}

func newGoogleGA4SDKClient(ctx context.Context, opts ...option.ClientOption) (*googleGA4SDKClient, error) {
	client, err := admin.NewAnalyticsAdminClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &googleGA4SDKClient{client: client}, nil
}

func (c *googleGA4SDKClient) ListProperties(ctx context.Context, account string) ([]GA4Property, error) {
	iter := c.client.ListProperties(ctx, &adminpb.ListPropertiesRequest{
		Filter:   "parent:" + account,
		PageSize: 200,
	})
	var out []GA4Property
	for {
		property, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, GA4Property{Name: property.GetName(), DisplayName: property.GetDisplayName()})
	}
}

func (c *googleGA4SDKClient) CreateProperty(ctx context.Context, req GA4CreatePropertyRequest) (GA4Property, error) {
	property, err := c.client.CreateProperty(ctx, &adminpb.CreatePropertyRequest{Property: &adminpb.Property{
		Parent:       req.Account,
		DisplayName:  req.DisplayName,
		TimeZone:     req.TimeZone,
		CurrencyCode: req.Currency,
	}})
	if err != nil {
		return GA4Property{}, err
	}
	return GA4Property{Name: property.GetName(), DisplayName: property.GetDisplayName()}, nil
}

func (c *googleGA4SDKClient) ListDataStreams(ctx context.Context, property string) ([]GA4DataStream, error) {
	iter := c.client.ListDataStreams(ctx, &adminpb.ListDataStreamsRequest{
		Parent:   property,
		PageSize: 200,
	})
	var out []GA4DataStream
	for {
		stream, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		web := stream.GetWebStreamData()
		out = append(out, GA4DataStream{
			Name:          stream.GetName(),
			DisplayName:   stream.GetDisplayName(),
			DefaultURI:    web.GetDefaultUri(),
			MeasurementID: web.GetMeasurementId(),
		})
	}
}

func (c *googleGA4SDKClient) CreateWebDataStream(ctx context.Context, req GA4CreateWebDataStreamRequest) (GA4DataStream, error) {
	stream, err := c.client.CreateDataStream(ctx, &adminpb.CreateDataStreamRequest{
		Parent: req.Property,
		DataStream: &adminpb.DataStream{
			Type:        adminpb.DataStream_WEB_DATA_STREAM,
			DisplayName: req.DisplayName,
			StreamData:  &adminpb.DataStream_WebStreamData_{WebStreamData: &adminpb.DataStream_WebStreamData{DefaultUri: req.DefaultURI}},
		},
	})
	if err != nil {
		return GA4DataStream{}, err
	}
	web := stream.GetWebStreamData()
	return GA4DataStream{
		Name:          stream.GetName(),
		DisplayName:   stream.GetDisplayName(),
		DefaultURI:    web.GetDefaultUri(),
		MeasurementID: web.GetMeasurementId(),
	}, nil
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
