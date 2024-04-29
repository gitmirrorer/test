# test
[![PkgGoDev](https://img.shields.io/badge/pkg.go.dev-reference-blue)](https://pkg.go.dev/github.com/gitmirrorer/test)
---

Use gitea for testing git logic.

## Usage

```go
package main_test

import (
	"context"
	"testing"
	"time"

	"github.com/gitmirrorer/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestClone(t *testing.T) {
	ctx := context.Background()
	server, err := test.ListenAndServe(ctx, "")
	require.NoError(t, err)
	defer server.Close(ctx)

	prep, err := test.PrepareRepositoriesOnAServer(
		context.Background(),
		server,
		t.Name(),
		map[string]string{
			"file1.txt": "File 1 Contents",
		},
		map[string]string{
			"file2.txt": "File 2 Contents",
		},
	)
	require.NoError(t, err)
	defer prep.Close(ctx)

	// work with the repositories
	// Clone(prep.Source.HTTP.URL)
}
```
