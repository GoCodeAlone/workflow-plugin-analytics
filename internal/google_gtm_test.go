package internal

import (
	"context"
	"testing"
)

func TestEnsureGTMDryRunPlansCreates(t *testing.T) {
	result, err := EnsureGTMWebContainer(context.Background(), nil, GTMEnsureRequest{
		Account:       "accounts/456",
		ContainerName: "example.com",
		Domains:       []string{"example.com"},
		WorkspaceName: "workflow",
		MeasurementID: "G-ABC123",
		DryRun:        true,
	})
	if err != nil {
		t.Fatalf("EnsureGTMWebContainer: %v", err)
	}
	if got := operationNames(result.Operations); !sameStrings(got, []string{"create_container", "create_workspace", "create_gtag_config"}) {
		t.Fatalf("operations = %v", got)
	}
}

func TestEnsureGTMReusesExistingResources(t *testing.T) {
	client := &fakeTagManagerClient{
		containers: []GTMContainer{{Path: "accounts/456/containers/1", Name: "example.com", PublicID: "GTM-ABC", Domains: []string{"example.com"}}},
		workspaces: map[string][]GTMWorkspace{
			"accounts/456/containers/1": {{Path: "accounts/456/containers/1/workspaces/2", Name: "workflow"}},
		},
		gtagConfigs: map[string][]GTMGtagConfig{
			"accounts/456/containers/1/workspaces/2": {{Path: "accounts/456/containers/1/workspaces/2/gtag_config/3", MeasurementID: "G-ABC123"}},
		},
	}
	result, err := EnsureGTMWebContainer(context.Background(), client, GTMEnsureRequest{
		Account:       "accounts/456",
		ContainerName: "example.com",
		Domains:       []string{"example.com"},
		WorkspaceName: "workflow",
		MeasurementID: "G-ABC123",
	})
	if err != nil {
		t.Fatalf("EnsureGTMWebContainer: %v", err)
	}
	if result.ContainerPath != "accounts/456/containers/1" || result.PublicID != "GTM-ABC" || result.WorkspacePath == "" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if client.createdContainers != 0 || client.createdWorkspaces != 0 || client.createdGtagConfigs != 0 {
		t.Fatalf("created container=%d workspace=%d config=%d", client.createdContainers, client.createdWorkspaces, client.createdGtagConfigs)
	}
}

func TestEnsureGTMCreatesMissingResources(t *testing.T) {
	client := &fakeTagManagerClient{}
	result, err := EnsureGTMWebContainer(context.Background(), client, GTMEnsureRequest{
		Account:       "accounts/456",
		ContainerName: "example.com",
		Domains:       []string{"example.com"},
		WorkspaceName: "workflow",
		MeasurementID: "G-ABC123",
	})
	if err != nil {
		t.Fatalf("EnsureGTMWebContainer: %v", err)
	}
	if result.ContainerPath == "" || result.PublicID == "" || result.WorkspacePath == "" || result.GtagConfigPath == "" {
		t.Fatalf("missing created IDs: %#v", result)
	}
}

func TestEnsureGTMRejectsInvalidInput(t *testing.T) {
	_, err := EnsureGTMWebContainer(context.Background(), nil, GTMEnsureRequest{
		Account:       "bad",
		ContainerName: "example.com",
		Domains:       []string{"not a host"},
		DryRun:        true,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

type fakeTagManagerClient struct {
	containers         []GTMContainer
	workspaces         map[string][]GTMWorkspace
	gtagConfigs        map[string][]GTMGtagConfig
	createdContainers  int
	createdWorkspaces  int
	createdGtagConfigs int
}

func (f *fakeTagManagerClient) ListContainers(_ context.Context, account string) ([]GTMContainer, error) {
	return f.containers, nil
}

func (f *fakeTagManagerClient) CreateWebContainer(_ context.Context, req GTMCreateContainerRequest) (GTMContainer, error) {
	f.createdContainers++
	c := GTMContainer{Path: req.Account + "/containers/created", Name: req.Name, PublicID: "GTM-CREATED", Domains: req.Domains}
	f.containers = append(f.containers, c)
	return c, nil
}

func (f *fakeTagManagerClient) ListWorkspaces(_ context.Context, containerPath string) ([]GTMWorkspace, error) {
	if f.workspaces == nil {
		return nil, nil
	}
	return f.workspaces[containerPath], nil
}

func (f *fakeTagManagerClient) CreateWorkspace(_ context.Context, req GTMCreateWorkspaceRequest) (GTMWorkspace, error) {
	f.createdWorkspaces++
	w := GTMWorkspace{Path: req.ContainerPath + "/workspaces/created", Name: req.Name}
	if f.workspaces == nil {
		f.workspaces = make(map[string][]GTMWorkspace)
	}
	f.workspaces[req.ContainerPath] = append(f.workspaces[req.ContainerPath], w)
	return w, nil
}

func (f *fakeTagManagerClient) ListGtagConfigs(_ context.Context, workspacePath string) ([]GTMGtagConfig, error) {
	if f.gtagConfigs == nil {
		return nil, nil
	}
	return f.gtagConfigs[workspacePath], nil
}

func (f *fakeTagManagerClient) CreateGtagConfig(_ context.Context, req GTMCreateGtagConfigRequest) (GTMGtagConfig, error) {
	f.createdGtagConfigs++
	c := GTMGtagConfig{Path: req.WorkspacePath + "/gtag_config/created", MeasurementID: req.MeasurementID}
	if f.gtagConfigs == nil {
		f.gtagConfigs = make(map[string][]GTMGtagConfig)
	}
	f.gtagConfigs[req.WorkspacePath] = append(f.gtagConfigs[req.WorkspacePath], c)
	return c, nil
}
