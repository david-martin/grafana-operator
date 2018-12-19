# Grafana Operator

*Status: Experimental*

A Grafana Operator as part of the application monitoring POC. Installs Grafana and syncs dashboards in the watched namespaces.

## Running locally

Make sure you are logged into your cluster using `oc` or `kubectl`.

1. Run the operator:

```sh
$ operator-sdk up local --namespace=application-monitoring
```

(assuming `application-monitoring` is the namespace where you have Grafana and Prometheus)