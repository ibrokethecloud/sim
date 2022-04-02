package etcd

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

// TestRunEmbeddedEtcd will run an embedded ETCD server and try to create and read a kv pair
func TestRunEmbeddedEtcd(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "etcd-")
	if err != nil {
		t.Fatalf("error creating etcd temp directory %v", err)
	}

	defer os.RemoveAll(dir)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// run an insecure etcd server
	e, err := RunEmbeddedEtcd(ctx, dir, nil)
	if err != nil {
		t.Fatalf("error running embedded etcd server %v", err)
	}

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   e.Endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("error creating etcd client %v", err)
	}
	defer etcdClient.Close()

	// put a value
	_, err = etcdClient.Put(ctx, "test", "true")
	if err != nil {
		t.Fatalf("error putting key test into etcd %v", err)
	}

	// get the value
	resp, err := etcdClient.Get(ctx, "test")
	if err != nil {
		t.Fatalf("error fetching key test from etcd %v", err)
	}

	if len(resp.Kvs) != 1 {
		t.Fatalf("expected to find 1 key-value but got %d", len(resp.Kvs))
	}

	for _, kv := range resp.Kvs {
		if string(kv.Key) != "test" {
			t.Fatalf("expected key test got %s", kv.Key)
		}
		if string(kv.Value) != "true" {
			t.Fatalf("expected key test got %s", kv.Key)
		}
	}
}
