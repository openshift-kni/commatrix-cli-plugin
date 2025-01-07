package cmd

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/openshift-kni/commatrix/pkg/types"
	mock_utils "github.com/openshift-kni/commatrix/pkg/utils/mock"
	"github.com/stretchr/testify/assert"
)

func Test_detectDeploymentAndInfra(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUtils := mock_utils.NewMockUtilsInterface(ctrl)
	t.Run("Detects SNO and Baremetal", func(t *testing.T) {
		mockUtils.EXPECT().IsSNOCluster().Return(true, nil)
		mockUtils.EXPECT().IsBMInfra().Return(true, nil)
		deployment, infra, err := detectDeploymentAndInfra(mockUtils)
		assert.NoError(t, err)
		assert.Equal(t, types.SNO, deployment)
		assert.Equal(t, types.Baremetal, infra)
	})
	t.Run("Detects Standard and Cloud", func(t *testing.T) {
		mockUtils.EXPECT().IsSNOCluster().Return(false, nil)
		mockUtils.EXPECT().IsBMInfra().Return(false, nil)
		deployment, infra, err := detectDeploymentAndInfra(mockUtils)
		assert.NoError(t, err)
		assert.Equal(t, types.Standard, deployment)
		assert.Equal(t, types.Cloud, infra)
	})

	t.Run("Fails on SNO detection error", func(t *testing.T) {
		mockUtils.EXPECT().IsSNOCluster().Return(false, errors.New("SNO detection failed"))
		_, _, err := detectDeploymentAndInfra(mockUtils)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to detect deployment type")
	})
	t.Run("Fails on Baremetal detection error", func(t *testing.T) {
		mockUtils.EXPECT().IsSNOCluster().Return(false, nil)
		mockUtils.EXPECT().IsBMInfra().Return(false, errors.New("BM detection failed"))
		_, _, err := detectDeploymentAndInfra(mockUtils)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to detect infra type")
	})
}
