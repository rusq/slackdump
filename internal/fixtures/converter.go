package fixtures

import _ "embed"

var (
	//go:embed assets/converter/chan1.json
	ConvertPublic1AllMessagesJSON string
	//go:embed assets/converter/chan1_thread_1736478630.905399.json
	ConvertPublic1AllThreadMessagesJSON string
)
