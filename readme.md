# kube-safe: maxpodcount

This project is Admission webhook-based fix for Kubernetes 1.10-bug that causes cluster to overuse it's resources if **Job** fails.

### Logic

```
1. Intercept all admission webhooks related to Pods
2. If the request is related to anything else than Job-related pod just accept and return
3. Check the count of pods related to the Job
  - If less than Job.Spec.BackoffLimit: accept and return
  - Else: Deny and return

```

This will cause Job to stop respawning new pods but will also freeze the job after spawning X-amount of failed pods.
All pods are still there and must be manually deleted.


### Installation
```
kubectl apply -f deploy.yml

# When the deployment starts it will automatically register itself to cluster
```
