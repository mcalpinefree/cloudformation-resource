# CloudFormation Resource

This is a Concourse CI resource that controls the deployment of CloudFormation
stacks.

# Resource type
You need to add this as a resource type to use it in your pipeline.
```yaml
resource_types:
- name: cloudformation-resource
  type: docker-image
  source:
    repository: pipelineci/cloudformation-resource
```

## Source Configuration

* `name`: *Required.* The stack name

* `aws_access_key_id`: *Required.*

* `aws_secret_access_key`: *Required.*

* `region`: *Required.*

### Example

Resource configuration for a CloudFormation stack:

``` yaml
resources:
- name: my-stack
  type: cloudformation
  source: name: my-stack
    aws_access_key_id: AUDFDQ7CA7JO6U56EQFW
    aws_secret_access_key: VY1SazRkI8M1JEIIUnwmxzMhfjaIzZABNVcqanj8
    region: ap-southeast-2
```

Creating/updating the stack:

```yaml
jobs:
- name: update-stack
  plan:
    - get: repo
      trigger: true
    - put: my-stack
      params:
        template: repo/my-stack.template
        parameters: repo/my-stack-parameters.json
```

Delete the stack:

```yaml
jobs:
- name: delete-stack
  plan:
    - put: my-stack
      params:
        delete: true
```

Get stack outputs:

```yaml
jobs:
- name: get-stack-outputs
  plan:
    - get: my-stack
```

## Behaviour

### `check`: Check for new stack events.

If a stack is updated or created this resource is triggered.

### `in`: Not implemented

There is no in behaviour implemented.

### `out`: Modify stack.

Create, update or delete the stack.

#### Parameters

* `template`: *Required.* The path of the CloudFormation template.
* `parameters`: *Optional.* The path of the CloudFormation parameters.
* `tags`: *Optional.* The path of the CloudFormation tags.
* `capabilities`: *Optional.* List of capabilities.
* `delete`: *Optional.* Set to `true` to delete the stack.
* `wait`: *Optional.* Defaults to true. If false is set it will update/create
  the stack and not wait for it to complete.
