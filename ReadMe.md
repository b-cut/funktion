## Funktion Operator


This operator watches for `Subscription` resources. When a new Subscription is created then this operator will spin up a matching `Deployment` which consumes from some endpoint and typically invokes a function using HTTP.
 
### Funktion Resources

A `Subscription` is modelled as a Kubernetes `ConfigMap` with the label `kind.funktion.fabric8.io: "Subscription"`. A ConfigMap is used so that the entries inside the `ConfigMap` can be mounted as files inside the `Deployment`. For example this will typically involve storing the `funktion.yml` file or maybe a Spring Boot `application.properties` file inside the `ConfigMap` like [this example subscription](examples/subscription1.yml)

A `Connector` is generated [for every Camel Component](https://github.com/fabric8io/funktion/blob/master/connectors/) and each connector has an associated `ConfigMap` resource like [this example](https://github.com/fabric8io/funktion/blob/master/connectors/connector-timer/src/main/fabric8/timer-cm.yml) which uses the label `kind.funktion.fabric8.io: "Connector"`. The `Connector` stores the `Deployment` metadata, the `schema.yml` for editing the connectors endpoint URL and the `documentation.adoc` documentation for using the Connector.

So a `Connector` can have `0..N Subscriptions` associated with it. For those who know [Apache Camel](http://camel.apache.org/) this is like the relationship between a `Component` having `0..N Endpoints`.


For example we could have a Connector called `kafka` which knows how to produce and consume messages on [Apache Kafka](http://camel.apache.org/kafka.html) with the Connector containing the metadata of how to create a consumer, how to configure the kafka endpoint and the documetnation. Then a Subscription could be created for `kafka://cheese` to subscribe on the `cheese` topic and post messages to `http://foo/`.
 

Typically a number of `Connector` resources are shipped as a package; such as inside the [Red Hat iPaaS](https://github.com/redhat-ipaas) or as an app inside fabric8. Though a `Connector` can be created as part of the CD Pipeline by an expert Java developer who takes a Camel component and customizes it for use by `Funktion` or the `iPaaS`.

The collection of `Connector` resources installed in a kubernetes namespace creates the `integration palette ` thats seen by users in tools like CLI or web UIs.

Then a `Subscription` can be created at any time by users from a `Connector` with a custom configuration (e.g. choosing a particular queue or topic in a messaging system or a particular table in a database or folder in a file system).


### Using the CLI
   
You can get help on the available commands via:
    
    funktion
    
To list all the Connectors or Subscriptions try:
    
    funktion get connector
    funktion get subscription

or to save typing you can use:

     funktion get c
     funktion get s

You can delete a Connector or Subscription via:

    funktion delete connector foo
    funktion delete subscription bar

Or to remove all the Subscriptions or Connectors use `--all`

    funktion delete subscription --all

#### Subscribing

To create a new subscription for a connector try the following:

    funktion subscribe --from timer://bar?period=5000 --to http://foo/

This will generate a new Subscription which will result in a new Deployment being created and one or more Pods should spin up.
 
You can then view the logs of a subscription via `kubectl`
 
    kubectl logs -f nameOfSubscription[TAB]

#### Scaling a Subscription

If you want to stop a subscription type:

    kubectl scale --replicas=0 deployment nameOfSubscription

To start it again:

    kubectl scale --replicas=1 deployment nameOfSubscription
    
### Using kubectl directly

You can also create a Subscription using `kubectl` if you prefer:

    kubectl apply -f https://github.com/fabric8io/funktion-operator/blob/master/examples/subscription1.yml

You can view all the Connectors and Subscriptions via:

    kubectl get cm

Or delete them via

    kubectl delete cm nameOfConnectorOrSubscription
    
### Debugging
   
If you ever need to you can debug any `Subscription` as each Subscription matches a `Deployment` of one or more pods. So you can just debug that pod which typically is a regular Spring Boot and camel application.
   
Otherwise you can debug the pod thats exposing an HTTP endpoint using whatever the native debugger is; e.g. using Java or NodeJS or whatever.   