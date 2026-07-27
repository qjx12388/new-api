package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDefaultNewUserGroupTestState(t *testing.T, group string) {
	t.Helper()
	setupUserUpdateTestState(t)
	oldGroup := common.DefaultNewUserGroup
	common.DefaultNewUserGroup = group
	t.Cleanup(func() {
		common.DefaultNewUserGroup = oldGroup
	})
}

func insertAndLoadStoredGroup(t *testing.T, username, group string) string {
	t.Helper()
	user := &User{
		Username: username,
		Group:    group,
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, user.Insert(0))

	var stored User
	require.NoError(t, DB.Where("username = ?", username).First(&stored).Error)
	return stored.Group
}

func TestInsertAppliesDefaultNewUserGroup(t *testing.T) {
	setupDefaultNewUserGroupTestState(t, "vip")

	assert.Equal(t, "vip", insertAndLoadStoredGroup(t, "default-group-user", ""))
}

func TestInsertFallsBackToDefaultWhenDefaultNewUserGroupMissing(t *testing.T) {
	setupDefaultNewUserGroupTestState(t, "group-that-does-not-exist")

	assert.Equal(t, "default", insertAndLoadStoredGroup(t, "missing-group-user", ""))
}

func TestInsertUsesDefaultGroupWhenDefaultNewUserGroupEmpty(t *testing.T) {
	setupDefaultNewUserGroupTestState(t, "")

	assert.Equal(t, "default", insertAndLoadStoredGroup(t, "no-config-user", ""))
}

func TestInsertKeepsExplicitGroupOverDefaultNewUserGroup(t *testing.T) {
	setupDefaultNewUserGroupTestState(t, "vip")

	assert.Equal(t, "svip", insertAndLoadStoredGroup(t, "explicit-group-user", "svip"))
}
