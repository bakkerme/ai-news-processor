# CI Workflows

This project uses GitHub Actions for continuous integration and deployment. The workflows automate testing and Docker image building to ensure code quality and streamline deployment.

## Available Workflows

### Run Tests

The `run-tests.yml` workflow runs automated tests on all branches and pull requests.

**Triggers:**
- Push to any branch
- Pull request to any branch

**Actions:**
1. Checkout the repository
2. Set up Go environment
3. Run tests with the Go testing framework
4. Report test results

### Build and Push Docker Image

The `build-and-push.yml` workflow builds a Docker image and pushes it to the GitHub Container Registry.

**Triggers:**
- After successful completion of the Run Tests workflow on the main branch

**Actions:**
1. Checkout the repository
2. Set up Docker Buildx
3. Login to GitHub Container Registry
4. Build and push the Docker image with appropriate tags

## Workflow Dependencies

The Docker image build workflow is configured to depend on the successful completion of the Run Tests workflow, ensuring that only tested code is deployed.

## Using the Docker Images

The Docker images built by the CI pipeline are available in the GitHub Container Registry. You can pull the latest image using:

```bash
docker pull ghcr.io/bakkerme/ai-news-processor:latest
```

You can also pull specific versions by tag.

## Local Testing

To run the same tests locally that are used in the CI pipeline:

```bash
go test ./...
```

This helps ensure that your changes will pass the CI pipeline before you push them. 