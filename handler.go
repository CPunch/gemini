package gemini

/* ======================================[[ pathHandler ]]======================================= */

type pathHandler struct {
	pathTbl map[string]func(peer *GeminiPeer)
}

func NewHandler() *pathHandler {
	return &pathHandler{pathTbl: map[string]func(peer *GeminiPeer){}}
}

func (pHndlr *pathHandler) AddHandler(path string, handler func(peer *GeminiPeer)) {
	pHndlr.pathTbl[path] = handler
}

func (pHndlr *pathHandler) HandlePeer(peer *GeminiPeer) {
	if hndlr, exists := pHndlr.pathTbl[peer.path]; exists {
		hndlr(peer)
	} else {
		peer.SendError("Path '" + peer.path + "' not found!")
	}
}
