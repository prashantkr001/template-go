# Minimum Viable Architecture

The default feature implemented in this template is trivially simple, though it conveys all the ideas we're trying to follow. It has 2 APIs, one to "Create an Item" and another to "List the items".

Following documentation would attempt at making it clear on how it implements MVA.

In the MVA, it is mentioned that we should follow the principles of I&I(Independence & Isolation) as much as practically possible, at all levels.

This template attempts at doing that by following the principles of [**D**omain **D**riven **D**esign/**D**evelopment](https://en.wikipedia.org/wiki/Domain-driven_design) & [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html).

Consider the 3 basic blocks defined in the MVA; I/O, Features & Dependencies. This repository has the following code structure which is inline with that.

## I/O

All the Input & Output of the application, are maintained within the `cmd` directory; further organized based on the communication protocol (server - http, subscriber - gcppubsub, kafka, cli - for command line etc.). There's a seemingly redundant package/layer called "API" in this template, which is used only to ensure that all the APIs provided by the application are consistent across all the communication protocols. Guiding the developers to maintain consistency with an obvious/explicit code structure. And the isolation is not just in the code structure, even the program control flow as well as dependencies are maintained in the same way. i.e. none of the internal packages are _aware_ of the APIs or the handlers for HTTP, Kafka etc. So the internals of the application are completely oblivious about the existence of the I/O block as mentioned in the MVA. Just like HTTP handlers, the Kafka subscriber/consumer is not a dependency of the features block. Hence there's no `consumer` interface defined, _*unlike*_ the `publisher` interface.

## Features & dependencies

The application features are maintained within the `internal` directory, further divided into _domain specific packages_. Trying to follow the principles of DDD, all the implementation required for the domain specific features are within the same package. The special `pkg` directory is maintained for generic code (e.g. database drivers, utility packages etc.) which are not specific or unique to a domain.

The _contracts_ for dependencies of features are defined as interfaces. In this template, you can see the same under:

```golang
// internal/item/store.go
// the interface for database dependency
type persistentStore interface{
    InsertItem(ctx context.Context, item Item) (*Item, error)
    ListItems(ctx context.Context, limit int) ([]Item, error)
    Item(ctx context.Context, id int) (*Item, error)
}

// internal/item/publisher.go
// the interface for queue dependency
type publisher interface {
    Publish(ctx context.Context, item *Item) error
}
```

A concrete example of how the dependencies are not tightly coupled with the feature is shown in the unit tests; where the dependencies are _easily_ mocked. The `TestInsertItem` function shows the entire usage of mocks, and how we can test _business features_ without the tight coupling with dependencies.

```golang
// internal/item/item_test.go
func TestInsertItem(t *testing.T) {}
```

### Note

It is important to remember that, `main.go` and the main package in general, would seem to break all the principles, and seem _ugly_. This is by design, and very much Go specific. Since main is the entry point for Go applications, it has to take care of initializations for every single package.
