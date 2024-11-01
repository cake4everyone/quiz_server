WS_PORT = $(shell yq .webserver.port config.yaml)
WS_PW = $(shell yq .webserver.password config.yaml)
WS_HOST = localhost

.PHONY: start update_questions_local update_questions

# start the quiz server
start:
	@go run main.go

# request to update all questions locally
update_questions_local: update_questions_request
# request to update all questions on the live server
update_questions: WS_HOST = api.cake4everyone.de
update_questions: update_questions_request

update_questions_request:
	@echo Fetching questions on '$(WS_HOST)' ... 
	@curl -X PUT $(WS_HOST):$(WS_PORT)/questions/fetch -d "{\"password\":\"$(WS_PW)\"}"
	@echo done!
