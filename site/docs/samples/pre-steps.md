# Tools used by the actors

- The data owner would typically register the dataset in a proprietary or open source catalog.  We have provided [katalog](../reference/katalog.md) - a thin layer acting as a replacement for the data catalog for evaluation purposes. This simplifies the sample deployment.
- The data owner stores credentials for accessing the dataset in kubernetes secrets.
- Proprietary and open source data governance systems are available either as part of a data catalog or as stand-alone systems.  This sample uses the open source [OpenPolicyAgent](https://www.openpolicyagent.org/).  The data governance officer writes the policies in OPA's [rego](https://www.openpolicyagent.org/docs/latest/policy-language/#what-is-rego) language.
- Any editor can be used to write the FybrikApplication.yaml via which the data user expresses the data usage requirements.
- A jupyter notebook is the workload from which the data is consumed by the data user.
- A Web Browser

# Prepare Fybrik environment

Typically, this would be done by an IT administrator.

- Install Fybrik using the [Quick Start](../get-started/quickstart.md) guide.
  This sample assumes the use of the built-in catalog, Open Policy Agent (OPA) and flight module.

# Create a namespace for the sample

Create a new Kubernetes namespace and set it as the active namespace:

```bash
kubectl create namespace fybrik-notebook-sample
kubectl config set-context --current --namespace=fybrik-notebook-sample
```

This enables easy [cleanup](../samples/cleanup.md) once you're done experimenting with the sample.
