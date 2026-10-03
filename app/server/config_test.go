package server

import (
	"reflect"
	"testing"
)

func TestTrustedProxyAddressesAcceptOnlyIPsAndCIDRs(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_IPS", " 127.0.0.1, 2001:db8::/32, invalid, , 10.2.0.3 ")
	want := []string{"127.0.0.1", "2001:db8::/32", "10.2.0.3"}
	if got := trustedProxyAddresses(); !reflect.DeepEqual(got, want) {
		t.Fatalf("trustedProxyAddresses() = %#v, want %#v", got, want)
	}
}

func TestTrustedProxyAddressesCanBeEmpty(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_IPS", "")
	if got := trustedProxyAddresses(); len(got) != 0 {
		t.Fatalf("trustedProxyAddresses() = %#v, want empty", got)
	}
}
