package valueset

import "fmt"

type ErrMultipleValues struct {
	Query   string
	TableId string
}

func (e ErrMultipleValues) Error() string {
	return fmt.Sprintf("Multiple values returned for lookup: %s, query: %s", string(e.TableId), e.Query)
}
