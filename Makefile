test:
	go test ./tests -check.v -test.v

embed:
ifeq (,$(wildcard ./wzbox))
	go get -v github.com/infra-whizz/wzbox
	go build -v -o wzbox github.com/infra-whizz/wzbox/cmd
endif
	./wzbox -f ./nanorunners/wrappers/pce.py -s WzPyPce -p nanocms_runners -c > ./nanorunners/pce.go

clean:
ifneq (wzbox,)
	rm wzbox
endif
