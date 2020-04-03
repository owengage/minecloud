package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractUniverse(t *testing.T) {
	dir := "/a/b/c/d"
	uni, world, err := getUniverseAndWorld(dir)
	assert.Equal(t, "/a/b/c", uni)
	assert.Equal(t, "d", world)
	assert.NoError(t, err)
}

func TestExtractUniverseWithRelative(t *testing.T) {
	dir := "a/b/c/d"
	uni, world, err := getUniverseAndWorld(dir)
	assert.Equal(t, "a/b/c", uni)
	assert.Equal(t, "d", world)
	assert.NoError(t, err)
}

func TestExtractUniverseWithRoot(t *testing.T) {
	dir := "/"
	_, _, err := getUniverseAndWorld(dir)
	assert.Error(t, err)
}

func TestExtractUniverseWithWorldAtRoot(t *testing.T) {
	dir := "/world"
	uni, world, err := getUniverseAndWorld(dir)
	assert.Equal(t, "/", uni)
	assert.Equal(t, "world", world)
	assert.NoError(t, err)
}

func TestExtractUniverseWithNothingErrors(t *testing.T) {
	dir := ""
	_, _, err := getUniverseAndWorld(dir)
	assert.Error(t, err)
}

func TestExtractUniverseWithEndingSlashIgnores(t *testing.T) {
	dir := "/world/"
	uni, world, err := getUniverseAndWorld(dir)
	assert.Equal(t, "/", uni)
	assert.Equal(t, "world", world)
	assert.NoError(t, err)
}

func TestExtractUniverseRelativeWithSlashAtEnd(t *testing.T) {
	dir := "world/"
	uni, world, err := getUniverseAndWorld(dir)
	assert.Equal(t, ".", uni)
	assert.Equal(t, "world", world)
	assert.NoError(t, err)
}
