test:
	go test ./tests -check.v -test.v

update: embed clean

embed:
ifeq (,$(wildcard ./wzbox))
	go get -v github.com/infra-whizz/wzbox
	go build -v -o wzbox github.com/infra-whizz/wzbox/cmd
endif
	python3 ./nanorunners/wrappers/stripsrc.py ./nanorunners/wrappers/pce.py > pce
	./wzbox -f pce -s WzPyPce -p nanocms_callers -c > ./nanorunners/callers/pce.go

clean:
ifneq (wzbox,)
	rm wzbox
endif
ifneq (pce,)
	rm pce
endif
