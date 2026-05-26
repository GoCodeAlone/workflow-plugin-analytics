package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/api/option"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

type GTMEnsureRequest struct {
	Account       string   `json:"account"`
	ContainerName string   `json:"container_name"`
	Domains       []string `json:"domains"`
	WorkspaceName string   `json:"workspace_name"`
	MeasurementID string   `json:"measurement_id,omitempty"`
	DryRun        bool     `json:"dry_run"`
}

type GTMEnsureResult struct {
	Account        string      `json:"account"`
	ContainerPath  string      `json:"container_path"`
	ContainerID    string      `json:"container_id"`
	PublicID       string      `json:"public_id"`
	WorkspacePath  string      `json:"workspace_path"`
	GtagConfigPath string      `json:"gtag_config_path,omitempty"`
	DryRun         bool        `json:"dry_run"`
	Operations     []Operation `json:"operations"`
}

type GTMContainer struct {
	Path     string
	ID       string
	Name     string
	PublicID string
	Domains  []string
}

type GTMWorkspace struct {
	Path string
	Name string
}

type GTMGtagConfig struct {
	Path          string
	MeasurementID string
}

type GTMCreateContainerRequest struct {
	Account string
	Name    string
	Domains []string
}

type GTMCreateWorkspaceRequest struct {
	ContainerPath string
	Name          string
}

type GTMCreateGtagConfigRequest struct {
	WorkspacePath string
	MeasurementID string
}

type TagManagerClient interface {
	ListContainers(ctx context.Context, account string) ([]GTMContainer, error)
	CreateWebContainer(ctx context.Context, req GTMCreateContainerRequest) (GTMContainer, error)
	ListWorkspaces(ctx context.Context, containerPath string) ([]GTMWorkspace, error)
	CreateWorkspace(ctx context.Context, req GTMCreateWorkspaceRequest) (GTMWorkspace, error)
	ListGtagConfigs(ctx context.Context, workspacePath string) ([]GTMGtagConfig, error)
	CreateGtagConfig(ctx context.Context, req GTMCreateGtagConfigRequest) (GTMGtagConfig, error)
}

func EnsureGTMWebContainer(ctx context.Context, client TagManagerClient, req GTMEnsureRequest) (GTMEnsureResult, error) {
	req.ContainerName = strings.TrimSpace(req.ContainerName)
	req.WorkspaceName = defaultString(req.WorkspaceName, "workflow")
	req.MeasurementID = strings.TrimSpace(req.MeasurementID)
	if err := validateGoogleAccount(req.Account); err != nil {
		return GTMEnsureResult{}, err
	}
	if req.ContainerName == "" {
		return GTMEnsureResult{}, fmt.Errorf("container_name is required")
	}
	domains, err := normalizeDomains(req.Domains)
	if err != nil {
		return GTMEnsureResult{}, err
	}
	result := GTMEnsureResult{Account: req.Account, DryRun: req.DryRun}

	var container GTMContainer
	if !req.DryRun {
		if client == nil {
			return GTMEnsureResult{}, fmt.Errorf("google credentials are required for live GTM ensure")
		}
		containers, err := client.ListContainers(ctx, req.Account)
		if err != nil {
			return result, err
		}
		for _, candidate := range containers {
			if candidate.Name == req.ContainerName || sameDomainSet(candidate.Domains, domains) {
				container = candidate
				result.Operations = append(result.Operations, reused("reuse_container"))
				break
			}
		}
	}
	if container.Path == "" {
		result.Operations = append(result.Operations, planned("create_container", req.DryRun))
		if req.DryRun {
			result.Operations = append(result.Operations, planned("create_workspace", true))
			if req.MeasurementID != "" {
				result.Operations = append(result.Operations, planned("create_gtag_config", true))
			}
			return result, nil
		}
		created, err := client.CreateWebContainer(ctx, GTMCreateContainerRequest{Account: req.Account, Name: req.ContainerName, Domains: domains})
		if err != nil {
			return result, err
		}
		container = created
	}
	result.ContainerPath = container.Path
	result.ContainerID = container.ID
	result.PublicID = container.PublicID

	var workspace GTMWorkspace
	workspaces, err := client.ListWorkspaces(ctx, container.Path)
	if err != nil {
		return result, err
	}
	for _, candidate := range workspaces {
		if candidate.Name == req.WorkspaceName {
			workspace = candidate
			result.Operations = append(result.Operations, reused("reuse_workspace"))
			break
		}
	}
	if workspace.Path == "" {
		result.Operations = append(result.Operations, planned("create_workspace", false))
		created, err := client.CreateWorkspace(ctx, GTMCreateWorkspaceRequest{ContainerPath: container.Path, Name: req.WorkspaceName})
		if err != nil {
			return result, err
		}
		workspace = created
	}
	result.WorkspacePath = workspace.Path

	if req.MeasurementID != "" {
		configs, err := client.ListGtagConfigs(ctx, workspace.Path)
		if err != nil {
			return result, err
		}
		for _, candidate := range configs {
			if candidate.MeasurementID == req.MeasurementID {
				result.GtagConfigPath = candidate.Path
				result.Operations = append(result.Operations, reused("reuse_gtag_config"))
				return result, nil
			}
		}
		result.Operations = append(result.Operations, planned("create_gtag_config", false))
		created, err := client.CreateGtagConfig(ctx, GTMCreateGtagConfigRequest{WorkspacePath: workspace.Path, MeasurementID: req.MeasurementID})
		if err != nil {
			return result, err
		}
		result.GtagConfigPath = created.Path
	}
	return result, nil
}

type googleTagManagerSDKClient struct {
	service *tagmanager.Service
}

func newGoogleTagManagerSDKClient(ctx context.Context, opts ...option.ClientOption) (*googleTagManagerSDKClient, error) {
	service, err := tagmanager.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &googleTagManagerSDKClient{service: service}, nil
}

func (c *googleTagManagerSDKClient) ListContainers(ctx context.Context, account string) ([]GTMContainer, error) {
	resp, err := c.service.Accounts.Containers.List(account).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	var out []GTMContainer
	for _, item := range resp.Container {
		out = append(out, GTMContainer{
			Path:     item.Path,
			ID:       item.ContainerId,
			Name:     item.Name,
			PublicID: item.PublicId,
			Domains:  item.DomainName,
		})
	}
	return out, nil
}

func (c *googleTagManagerSDKClient) CreateWebContainer(ctx context.Context, req GTMCreateContainerRequest) (GTMContainer, error) {
	item, err := c.service.Accounts.Containers.Create(req.Account, &tagmanager.Container{
		Name:         req.Name,
		DomainName:   req.Domains,
		UsageContext: []string{"web"},
	}).Context(ctx).Do()
	if err != nil {
		return GTMContainer{}, err
	}
	return GTMContainer{Path: item.Path, ID: item.ContainerId, Name: item.Name, PublicID: item.PublicId, Domains: item.DomainName}, nil
}

func (c *googleTagManagerSDKClient) ListWorkspaces(ctx context.Context, containerPath string) ([]GTMWorkspace, error) {
	resp, err := c.service.Accounts.Containers.Workspaces.List(containerPath).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	var out []GTMWorkspace
	for _, item := range resp.Workspace {
		out = append(out, GTMWorkspace{Path: item.Path, Name: item.Name})
	}
	return out, nil
}

func (c *googleTagManagerSDKClient) CreateWorkspace(ctx context.Context, req GTMCreateWorkspaceRequest) (GTMWorkspace, error) {
	item, err := c.service.Accounts.Containers.Workspaces.Create(req.ContainerPath, &tagmanager.Workspace{Name: req.Name}).Context(ctx).Do()
	if err != nil {
		return GTMWorkspace{}, err
	}
	return GTMWorkspace{Path: item.Path, Name: item.Name}, nil
}

func (c *googleTagManagerSDKClient) ListGtagConfigs(ctx context.Context, workspacePath string) ([]GTMGtagConfig, error) {
	resp, err := c.service.Accounts.Containers.Workspaces.GtagConfig.List(workspacePath).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	var out []GTMGtagConfig
	for _, item := range resp.GtagConfig {
		out = append(out, GTMGtagConfig{Path: item.Path, MeasurementID: gtagMeasurementID(item)})
	}
	return out, nil
}

func (c *googleTagManagerSDKClient) CreateGtagConfig(ctx context.Context, req GTMCreateGtagConfigRequest) (GTMGtagConfig, error) {
	item, err := c.service.Accounts.Containers.Workspaces.GtagConfig.Create(req.WorkspacePath, &tagmanager.GtagConfig{
		Type: "ga4",
		Parameter: []*tagmanager.Parameter{
			{Type: "template", Key: "measurementId", Value: req.MeasurementID},
		},
	}).Context(ctx).Do()
	if err != nil {
		return GTMGtagConfig{}, err
	}
	return GTMGtagConfig{Path: item.Path, MeasurementID: gtagMeasurementID(item)}, nil
}

func gtagMeasurementID(config *tagmanager.GtagConfig) string {
	for _, param := range config.Parameter {
		if param.Key == "measurementId" || param.Key == "measurement_id" {
			return param.Value
		}
	}
	return ""
}

func sameDomainSet(a, b []string) bool {
	aa, errA := normalizeDomains(a)
	bb, errB := normalizeDomains(b)
	if errA != nil || errB != nil || len(aa) != len(bb) {
		return false
	}
	sort.Strings(aa)
	sort.Strings(bb)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}
