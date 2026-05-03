// SPDX-License-Identifier: GPL-2.0 OR MIT
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/if_packet.h>
#include <linux/pkt_cls.h>
#include <linux/ip.h>
#include <bpf/bpf_helpers.h>

struct bridge_ports {
	__u32 tap_ifindex;
	__u32 veth_ifindex;
};

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, struct bridge_ports);
} bridge_cfg SEC(".maps");

SEC("tc")
int tc_l2_proxy(struct __sk_buff *ctx)
{
	__u32 k = 0;
	struct bridge_ports *cfg = bpf_map_lookup_elem(&bridge_cfg, &k);
	if (!cfg || cfg->tap_ifindex == 0 || cfg->veth_ifindex == 0)
		return TC_ACT_OK;

	int in_ifindex = ctx->ifindex;

	if (in_ifindex == cfg->tap_ifindex) {
		bpf_skb_change_type(ctx, PACKET_HOST);
		return bpf_redirect_peer(cfg->veth_ifindex, 0);
	}
	if (in_ifindex == cfg->veth_ifindex)
		return bpf_redirect(cfg->tap_ifindex, 0);

	return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
