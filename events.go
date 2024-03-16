package sdk

var (
	EventWSReconnected  = "websocket_reconnected"  //ws重新连上时的触发的事件
	EventWSDisconnected = "websocket_disconnected" //ws断开时的触发的事件

	// 以下为平台事件
	EventPong               = "pong"
	EventDialogCreated      = "chat_DialogCreated"
	EventMessageRead        = "chat_MessageRead"
	EventMessageSent        = "MessageSent"
	EventDialogLimitChanged = "chat_DialogLimitChanged"
)
