package export

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConfig_BuildClientConfig(t *testing.T) {
	// GIVEN a file that overrides some fields in the default configuration
	cfgFile, err := ioutil.TempFile("", "testload_")
	require.NoError(t, err)
	_, err = cfgFile.WriteString(`
tenantID: theTenant
url: "https://foo:8888/"
batchWait: 1m
minBackOff: 5s
labels:
  - foo
  - bar
staticLabels:
  baz: bae
  tiki: taka
printInput: true
`)
	require.NoError(t, err)
	require.NoError(t, cfgFile.Close())

	// WHEN it is loaded and transformed into a Loki Config
	cfg, err := LoadConfig(cfgFile.Name())
	require.NoError(t, err)
	assert.Contains(t, cfg.Labels, "foo")
	assert.Contains(t, cfg.Labels, "bar")
	assert.True(t, cfg.PrintInput)
	assert.EqualValues(t, "bae", cfg.StaticLabels["baz"])
	assert.EqualValues(t, "taka", cfg.StaticLabels["tiki"])

	// THEN the generated loki config is consistent with the parsed data
	ccfg, err := cfg.buildLokiConfig()
	require.NoError(t, err)

	require.NotNil(t, ccfg.URL)
	assert.Equal(t, "https://foo:8888/loki/api/v1/push", ccfg.URL.String())
	assert.Equal(t, "theTenant", ccfg.TenantID)
	assert.Equal(t, time.Minute, ccfg.BatchWait)
	assert.NotZero(t, ccfg.BatchSize)
	assert.Equal(t, cfg.BatchSize, ccfg.BatchSize)
	assert.Equal(t, cfg.MinBackoff, ccfg.BackoffConfig.MinBackoff)

	_ = mock.Mock{}
	assert.True(t, true)
	require.True(t, true)
}
