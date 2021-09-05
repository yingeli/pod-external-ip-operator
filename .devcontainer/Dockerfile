# [Choice] Go version: 1, 1.16, 1.15
ARG VARIANT=1.16
FROM mcr.microsoft.com/vscode/devcontainers/go:0-${VARIANT} as go

# You can pick any Debian/Ubuntu-based image. 😊
FROM go as go-az

# [Option] Install zsh
ARG INSTALL_ZSH="true"
# [Option] Upgrade OS packages to their latest versions
ARG UPGRADE_PACKAGES="false"

# Install needed packages and setup non-root user. Use a separate RUN statement to add your own dependencies.
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID
COPY azure-cli/library-scripts/*.sh /tmp/library-scripts/
RUN bash /tmp/library-scripts/common-debian.sh "${INSTALL_ZSH}" "${USERNAME}" "${USER_UID}" "${USER_GID}" "${UPGRADE_PACKAGES}" "true" "true" \
    # Install the Azure CLI
    && bash /tmp/library-scripts/azcli-debian.sh \
    # Clean up
    && apt-get clean -y && rm -rf /var/lib/apt/lists/* /tmp/library-scripts

# [Optional] Uncomment this section to install additional OS packages.
# RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
#     && apt-get -y install --no-install-recommends <your-package-list-here>


# Note: You can use any Debian/Ubuntu based image you want. 
FROM go-az as go-az-k8s

# Options
ARG INSTALL_ZSH="true"
ARG UPGRADE_PACKAGES="false"
ARG USE_MOBY="true"
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID
ARG ENABLE_NONROOT_DOCKER="true"
# Install needed packages and setup non-root user. Use a separate RUN statement to add your own dependencies.
COPY kubernetes-helm/library-scripts/*.sh /tmp/library-scripts/
RUN apt-get update && /bin/bash /tmp/library-scripts/common-debian.sh "${INSTALL_ZSH}" "${USERNAME}" "${USER_UID}" "${USER_GID}" "${UPGRADE_PACKAGES}" "true" "true" \
    #&& /bin/bash /tmp/library-scripts/docker-debian.sh "true" "/var/run/docker-host.sock" "/var/run/docker.sock" "${USERNAME}" "${USE_MOBY}" \
    && /bin/bash /tmp/library-scripts/docker-in-docker-debian.sh "${ENABLE_NONROOT_DOCKER}" "${USERNAME}" "${USE_MOBY}" \
    && /bin/bash /tmp/library-scripts/kubectl-helm-debian.sh "latest" "latest" "none" \
    && apt-get autoremove -y && apt-get clean -y && rm -rf /var/lib/apt/lists/* /tmp/library-scripts/

# Script copies localhost's ~/.kube/config file into the container and swaps out 
# localhost for host.docker.internal on bash/zsh start to keep them in sync.
COPY kubernetes-helm/copy-kube-config.sh /usr/local/share/
RUN chown ${USERNAME}:root /usr/local/share/copy-kube-config.sh \
    && echo "source /usr/local/share/copy-kube-config.sh" | tee -a /root/.bashrc /root/.zshrc /home/${USERNAME}/.bashrc >> /home/${USERNAME}/.zshrc

VOLUME [ "/var/lib/docker" ]

# Setting the ENTRYPOINT to docker-init.sh will configure non-root access to 
# the Docker socket if "overrideCommand": false is set in devcontainer.json. 
# The script will also execute CMD if you need to alter startup behaviors.
ENTRYPOINT [ "/usr/local/share/docker-init.sh" ]
CMD [ "sleep", "infinity" ]

# [Optional] Uncomment this section to install additional OS packages.
# RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
#     && apt-get -y install --no-install-recommends <your-package-list-here>

# download kubebuilder and install locally.
FROM go-az-k8s

RUN curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH) \
    && chmod +x kubebuilder && mv kubebuilder /usr/local/bin/