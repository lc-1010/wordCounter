package main

import (
	"testing"
)

func Test_readUrl(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "rust",
			args: args{
				url: "https://github.com/rust-lang/book/tree/main/src",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//rf := getUrl(tt.args.url)
			//fmt.Println(rf)
		})
	}
}
