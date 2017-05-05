## 依赖
* [Serf: Service orchestration and management tool. ](https://github.com/hashicorp/serf)
    * [官网](https://www.serf.io/) 
    * Discovering web servers and automatically adding them to a load balancer
    * Organizing many memcached or redis nodes into a cluster, perhaps with something like twemproxy or maybe just configuring an application with the address of all the nodes
    * Triggering web deploys using the event system built on top of Serf
    * Propagating changes to configuration to relevant nodes
    * Updating DNS records to reflect cluster changes as they occur

* [HashiCorp Raft](https://github.com/hashicorp/raft)