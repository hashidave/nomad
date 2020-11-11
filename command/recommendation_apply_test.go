package command

import (
	"fmt"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/testutil"
	"github.com/mitchellh/cli"
	"github.com/posener/complete"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecommendationApplyCommand_Run(t *testing.T) {
	require := require.New(t)
	t.Parallel()
	srv, client, url := testServer(t, true, nil)
	defer srv.Shutdown()
	testutil.WaitForResult(func() (bool, error) {
		nodes, _, err := client.Nodes().List(nil)
		if err != nil {
			return false, err
		}
		if len(nodes) == 0 {
			return false, fmt.Errorf("missing node")
		}
		if _, ok := nodes[0].Drivers["mock_driver"]; !ok {
			return false, fmt.Errorf("mock_driver not ready")
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})

	ui := cli.NewMockUi()
	cmd := &RecommendationApplyCommand{Meta: Meta{Ui: ui}}

	// Register a test job to write a recommendation against.
	testJob := testJob("recommendation_apply")
	regResp, _, err := client.Jobs().Register(testJob, nil)
	require.NoError(err)
	registerCode := waitForSuccess(ui, client, fullId, t, regResp.EvalID)
	require.Equal(0, registerCode)

	// Write a recommendation.
	rec := api.Recommendation{
		JobID:    *testJob.ID,
		Group:    *testJob.TaskGroups[0].Name,
		Task:     testJob.TaskGroups[0].Tasks[0].Name,
		Resource: "CPU",
		Value:    1,
		Meta:     map[string]interface{}{"test-meta-entry": "test-meta-value"},
		Stats:    map[string]float64{"p13": 1.13},
	}
	recResp, _, err := client.Recommendations().Upsert(&rec, nil)
	if srv.Enterprise {
		require.NoError(err)

		// Read the recommendation out to ensure it is there as a control on
		// later tests.
		recInfo, _, err := client.Recommendations().Info(recResp.ID, nil)
		require.NoError(err)
		require.NotNil(recInfo)
	} else {
		require.Error(err, "Nomad Enterprise only endpoint")
	}

	// Only perform the call if we are running enterprise tests. Otherwise the
	// recResp object will be nil.
	if !srv.Enterprise {
		return
	}
	code := cmd.Run([]string{"-address=" + url, recResp.ID})
	require.Equal(0, code)

	// Perform an info call on the recommendation which should return not
	// found.
	recInfo, _, err := client.Recommendations().Info(recResp.ID, nil)
	require.Error(err, "not found")
	require.Nil(recInfo)

	// Check the new jobspec to see if the resource value has changed.
	jobResp, _, err := client.Jobs().Info(*testJob.ID, nil)
	require.NoError(err)
	require.Equal(1, *jobResp.TaskGroups[0].Tasks[0].Resources.CPU)
}

func TestRecommendationApplyCommand_AutocompleteArgs(t *testing.T) {
	assert := assert.New(t)
	t.Parallel()

	srv, client, url := testServer(t, true, nil)
	defer srv.Shutdown()

	// Register a test job to write a recommendation against.
	ui := cli.NewMockUi()
	testJob := testJob("recommendation_list")
	regResp, _, err := client.Jobs().Register(testJob, nil)
	require.NoError(t, err)
	registerCode := waitForSuccess(ui, client, fullId, t, regResp.EvalID)
	require.Equal(t, 0, registerCode)

	// Write a recommendation.
	rec := &api.Recommendation{
		JobID:    *testJob.ID,
		Group:    *testJob.TaskGroups[0].Name,
		Task:     testJob.TaskGroups[0].Tasks[0].Name,
		Resource: "CPU",
		Value:    1050,
		Meta:     map[string]interface{}{"test-meta-entry": "test-meta-value"},
		Stats:    map[string]float64{"p13": 1.13},
	}
	rec, _, err = client.Recommendations().Upsert(rec, nil)
	if srv.Enterprise {
		require.NoError(t, err)
	} else {
		require.Error(t, err, "Nomad Enterprise only endpoint")
		return
	}

	cmd := &RecommendationApplyCommand{Meta: Meta{Ui: ui, flagAddress: url}}
	prefix := rec.ID[:5]
	args := complete.Args{Last: prefix}
	predictor := cmd.AutocompleteArgs()

	res := predictor.Predict(args)
	assert.Equal(1, len(res))
	assert.Equal(rec.ID, res[0])
}
