package batch

import (
	"context"
)

type ShortlinkBatchProcessor interface {
	BatchDeleteShortlinks(ctx context.Context, userUID string, linkUIDs []string)
}
