package pypa_test

import (
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/datawire/dlib/dlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/python/pep425"
	"github.com/datawire/ocibuild/pkg/python/pep440"
	"github.com/datawire/ocibuild/pkg/python/pypa/simple_repo_api"
)

func TestDownload(t *testing.T) {
	testDownloadedWheels(t, func(t *testing.T, filename string, content []byte) {})
}

func testDownloadedWheels(t *testing.T, fn func(t *testing.T, filename string, content []byte)) {
	type Download struct {
		Name             string
		Version          string
		ExpectedFilename string
		ExpectedSHA1Sum  string
	}
	// This list and the python version info are based on Emissary 2.0.5.
	//
	// Don't test 'setuptools' or 'pip'; pip contains special cases for them (that aren't worth
	// replicating), and so TestPIP will never pass on them.
	//
	//nolint:lll // big table
	downloads := []Download{
		{"Flask", "1.1.2", "Flask-1.1.2-py2.py3-none-any.whl", "fc9a504c245e5b2425c20da15012566a1f633e60"},
		{"GitPython", "3.1.11", "GitPython-3.1.11-py3-none-any.whl", "6be334a7292005d0a9505777d3ef0e9ef93e4cfe"},
		{"Jinja2", "2.11.2", "Jinja2-2.11.2-py2.py3-none-any.whl", "ecdb7ab7e70f2f39bfa05ea53f9ea7553dbfcf4b"},
		{"Werkzeug", "1.0.1", "Werkzeug-1.0.1-py2.py3-none-any.whl", "38fb5646b509481cda1f0e63d5dae050eb2875cb"},
		{"attrs", "19.3.0", "attrs-19.3.0-py2.py3-none-any.whl", "7861c8ac9909cde554a15db6494ddf268bc55f3c"},
		{"cachetools", "4.1.1", "cachetools-4.1.1-py3-none-any.whl", "3ac3c79a4cc5f1014b001b3ca345a601ec10a063"},
		{"certifi", "2020.6.20", "certifi-2020.6.20-py2.py3-none-any.whl", "f37eb16dd168e81f95c31495516aa948e7b846ee"},
		{"chardet", "3.0.4", "chardet-3.0.4-py2.py3-none-any.whl", "96faab7de7e9a71b37f22adb64daf2898e967e3e"},
		{"click", "7.1.2", "click-7.1.2-py2.py3-none-any.whl", "0a1cbd250993d47464454a761a53e8a394827d41"},
		{"clize", "4.1.1", "clize-4.1.1-py2.py3-none-any.whl", "86c93d5301b7519aec32bc223e8f0346ebfd1034"},
		{"decorator", "4.4.2", "decorator-4.4.2-py2.py3-none-any.whl", "f5d1e037f985e9878538e26467e90b0cb9a579ba"},
		{"docutils", "0.15.2", "docutils-0.15.2-py3-none-any.whl", "aafeddc912b74557754b2aaece3f1364be8e9f6a"},
		{"gitdb", "4.0.5", "gitdb-4.0.5-py3-none-any.whl", "d0701d6d90719ccf496953414be7c07f1edec4d0"},
		{"google_auth", "1.23.0", "google_auth-1.23.0-py2.py3-none-any.whl", "ede9ca2f8f9e9dbfcead51611f86e832acbd3d6c"},
		{"gunicorn", "20.0.4", "gunicorn-20.0.4-py2.py3-none-any.whl", "5b9580f6c90af9b2d97488e3d17143cca0b6de2a"},
		{"idna", "2.7", "idna-2.7-py2.py3-none-any.whl", "18abb4c2adda523d5a70d1c7c18d801acea402a0"},
		{"iniconfig", "1.1.1", "iniconfig-1.1.1-py2.py3-none-any.whl", "77378afaf23c05657bed196a0e359d3e6a3f1cf2"},
		{"itsdangerous", "1.1.0", "itsdangerous-1.1.0-py2.py3-none-any.whl", "daa770893bf76273c98266eab052ed2b06ff0474"},
		{"jsonpatch", "1.30", "jsonpatch-1.30-py2.py3-none-any.whl", "659279a0b5f0f85d88db3fc27b0391c185c20375"},
		{"jsonpointer", "2.0", "jsonpointer-2.0-py2.py3-none-any.whl", "08d023333acbd05bbd5a9549cb5957c57adcb816"},
		{"jsonschema", "3.2.0", "jsonschema-3.2.0-py2.py3-none-any.whl", "13a9abc0b85f73adfea760809110f4520118e1a4"},
		{"kubernetes", "9.0.0", "kubernetes-9.0.0-py2.py3-none-any.whl", "37590f91cbd219b20381fdf014d191cce0f92327"},
		{"mypy", "0.790", "mypy-0.790-py3-none-any.whl", "51e54c14e3627423d5d20107aad7bd7a1387aa72"},
		{"mypy_extensions", "0.4.3", "mypy_extensions-0.4.3-py2.py3-none-any.whl", "b17fb71194bd21b8ec20d711298d2197c5eb34d0"},
		{"oauthlib", "3.1.0", "oauthlib-3.1.0-py2.py3-none-any.whl", "52c42c870fea99ec71d4b1af876e3fb73f7c6ddf"},
		{"od", "1.0", "od-1.0-py3-none-any.whl", "908be03348e25a0ce937badd848c6df1e797a94c"},
		{"packaging", "20.4", "packaging-20.4-py2.py3-none-any.whl", "4047d7afbf467986709e0856a7526a9211d6e030"},
		{"pexpect", "4.8.0", "pexpect-4.8.0-py2.py3-none-any.whl", "0c3fc83ca045abeec9ce82bb7ee3e77f0390bca4"},
		{"pluggy", "0.13.1", "pluggy-0.13.1-py2.py3-none-any.whl", "bea186814eafe56aa821db73b242772329e60555"},
		{"prometheus_client", "0.9.0", "prometheus_client-0.9.0-py2.py3-none-any.whl", "1cf49baa5ae1e0dedde4bc5c9a46e5c47550168a"},
		{"protobuf", "3.13.0", "protobuf-3.13.0-py2.py3-none-any.whl", "0c2b63334e1f3ee96e48fede5e8ba68cc0c2eed4"},
		{"ptyprocess", "0.6.0", "ptyprocess-0.6.0-py2.py3-none-any.whl", "0510a31b1a72328147f3b17d33002a442a7e294c"},
		{"py", "1.9.0", "py-1.9.0-py2.py3-none-any.whl", "7e5839cc09f1470c93726c7f565e38a88c870f1c"},
		{"pyasn1", "0.4.8", "pyasn1-0.4.8-py2.py3-none-any.whl", "c3c9f195dc89eb6d04828b881314743b548318d0"},
		{"pyasn1_modules", "0.2.8", "pyasn1_modules-0.2.8-py2.py3-none-any.whl", "d77aa46abbcaccc4054a0777a191e427c785c65a"},
		{"pyparsing", "2.4.7", "pyparsing-2.4.7-py2.py3-none-any.whl", "c8307f47e3b75a2d02af72982a2dfefa3f56e407"},
		{"pytest", "6.1.2", "pytest-6.1.2-py3-none-any.whl", "a6d673d06b90d3c2877b90d01541a9aa1f011f7f"},
		{"pytest_cov", "2.10.1", "pytest_cov-2.10.1-py2.py3-none-any.whl", "5f4e7ea46611a7aa9b8e85070ffabd3fbb7c817a"},
		{"pytest_rerunfailures", "9.1.1", "pytest_rerunfailures-9.1.1-py3-none-any.whl", "365c8ecb163658dcb47763506dcc12525497bc3d"},
		{"python_dateutil", "2.8.1", "python_dateutil-2.8.1-py2.py3-none-any.whl", "3005ff67df93ee276fb8631e17c677df852254ad"},
		{"requests", "2.25.1", "requests-2.25.1-py2.py3-none-any.whl", "b1009d9fd6acadc64e1a3cecb6f0083fe047e753"},
		{"requests_oauthlib", "1.3.0", "requests_oauthlib-1.3.0-py2.py3-none-any.whl", "25d5667d7a61586f5ddaac7e08cc3053db3d8661"},
		{"retry", "0.9.2", "retry-0.9.2-py2.py3-none-any.whl", "d043af40ba7218e67e575747fc7da43c10101f00"},
		{"rsa", "4.6", "rsa-4.6-py3-none-any.whl", "61064ab969e05b52559a4c4b6a6409fc5a228fa6"},
		{"semantic_version", "2.8.5", "semantic_version-2.8.5-py2.py3-none-any.whl", "31a93c0da0fc93edf9f49285af349159e30c6e9f"},
		{"sigtools", "2.0.2", "sigtools-2.0.2-py2.py3-none-any.whl", "809f83e435cf8e44b4abf960ce8e8b094e5e7557"},
		{"six", "1.15.0", "six-1.15.0-py2.py3-none-any.whl", "8730d16507db66e828c696ecc7cb785e557900bb"},
		{"smmap", "3.0.4", "smmap-3.0.4-py2.py3-none-any.whl", "b72053c9a674095e1cf6d5d87bec1feace392a40"},
		{"toml", "0.10.2", "toml-0.10.2-py2.py3-none-any.whl", "a55ae166e643e6c7a28c16fe005efc32ee98ee76"},
		{"typing_extensions", "3.7.4.3", "typing_extensions-3.7.4.3-py3-none-any.whl", "0737ee9410a7c82511dd5e732092ce4ab66d51b4"},
		{"urllib3", "1.26.3", "urllib3-1.26.3-py2.py3-none-any.whl", "bc1f2e29068a85cefc6c7652ae77eea287e0c9d8"},
		{"websocket_client", "0.57.0", "websocket_client-0.57.0-py2.py3-none-any.whl", "9dc6541510a95e4d043b59294ebe3f5861d0f5a3"},
	}

	pythonVersion, err := pep440.ParseVersion("3.8.10")
	require.NoError(t, err)
	pythonTagStrs := []string{
		"cp38-cp38-linux_x86_64",
		"cp38-abi3-linux_x86_64",
		"cp38-none-linux_x86_64",
		"cp37-abi3-linux_x86_64",
		"cp36-abi3-linux_x86_64",
		"cp35-abi3-linux_x86_64",
		"cp34-abi3-linux_x86_64",
		"cp33-abi3-linux_x86_64",
		"cp32-abi3-linux_x86_64",
		"py38-none-linux_x86_64",
		"py3-none-linux_x86_64",
		"py37-none-linux_x86_64",
		"py36-none-linux_x86_64",
		"py35-none-linux_x86_64",
		"py34-none-linux_x86_64",
		"py33-none-linux_x86_64",
		"py32-none-linux_x86_64",
		"py31-none-linux_x86_64",
		"py30-none-linux_x86_64",
		"py38-none-any",
		"py3-none-any",
		"py37-none-any",
		"py36-none-any",
		"py35-none-any",
		"py34-none-any",
		"py33-none-any",
		"py32-none-any",
		"py31-none-any",
		"py30-none-any",
	}

	pythonTags := make(pep425.Installer, 0, len(pythonTagStrs))
	for _, str := range pythonTagStrs {
		parts := strings.Split(str, "-")
		require.Equal(t, 3, len(parts))
		pythonTags = append(pythonTags, pep425.Tag{
			Python:   parts[0],
			ABI:      parts[1],
			Platform: parts[2],
		})
	}

	client := simple_repo_api.NewClient(pythonVersion, pythonTags)
	t.Parallel()
	for _, testDownload := range downloads {
		testDownload := testDownload
		t.Run(testDownload.ExpectedFilename, func(t *testing.T) {
			t.Parallel()
			ctx := dlog.NewTestContext(t, true)
			specifier, err := pep440.ParseSpecifier("==" + testDownload.Version)
			require.NoError(t, err)
			require.NotNil(t, ctx)

			link, err := client.SelectWheel(ctx, testDownload.Name, specifier)
			require.NoError(t, err)
			require.NotNil(t, link)
			require.Equal(t, testDownload.ExpectedFilename, link.Text)

			u, err := url.Parse(link.HRef)
			require.NoError(t, err)
			require.NotNil(t, u)
			require.Equal(t, testDownload.ExpectedFilename, path.Base(u.Path))

			content, err := link.Get(ctx)
			require.NoError(t, err)
			sum := sha1.Sum(content)
			assert.Equal(t, testDownload.ExpectedSHA1Sum, hex.EncodeToString(sum[:]))

			if !t.Failed() {
				fn(t, link.Text, content)
			}
		})
	}
}
