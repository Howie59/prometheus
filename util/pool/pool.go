// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pool

import (
	"fmt"
	"reflect"
	"sync"
)

// Pool is a bucketed pool for variably sized byte slices.
// Pool 存储不同大小slice的桶
type Pool struct {
	// 每个bucket分区对应的sync.pool实例
	buckets []sync.Pool
	// 每个bucket分区中buffer缓冲区的大小
	sizes   []int
	// 如果分区中buffer缓冲区耗尽或者没有找到合适的bucket分区来获取，
	// 会通过这个函数新建buffer缓冲区并返回给调用方
	make func(int) interface{}
}

// factor为递增比例
func New(minSize, maxSize int, factor float64, makeFunc func(int) interface{}) *Pool {
	if minSize < 1 {
		panic("invalid minimum pool size")
	}
	if maxSize < 1 {
		panic("invalid maximum pool size")
	}
	if factor < 1 {
		panic("invalid factor")
	}

	var sizes []int

	// 计算每个bucket分区存储的buffer大小
	for s := minSize; s <= maxSize; s = int(float64(s) * factor) {
		sizes = append(sizes, s)
	}

	p := &Pool{
		buckets: make([]sync.Pool, len(sizes)),
		sizes:   sizes,
		make:    makeFunc,
	}

	return p
}

// Get returns a new byte slices that fits the given size.
func (p *Pool) Get(sz int) interface{} {
	for i, bktSize := range p.sizes {
		if sz > bktSize {
			continue
		}
		b := p.buckets[i].Get()
		if b == nil {
			b = p.make(bktSize)
		}
		return b
	}
	// 申请的buffer太大没有合适的bucket分区，直接通过make新建缓冲区
	return p.make(sz)
}

// Put adds a slice to the right bucket in the pool.
func (p *Pool) Put(s interface{}) {
	slice := reflect.ValueOf(s)

	if slice.Kind() != reflect.Slice {
		panic(fmt.Sprintf("%+v is not a slice", slice))
	}
	for i, size := range p.sizes {
		if slice.Cap() > size {
			continue
		}
		p.buckets[i].Put(slice.Slice(0, 0).Interface())
		return
	}
}
