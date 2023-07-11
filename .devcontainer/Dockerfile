FROM ghcr.io/lesomnus/dev-golang:1.20

ENV VCPKG_ROOT=/opt/vcpkg \
	VCPKG_DOWNLOADS=/opt/vcpkg-downloads \
	VCPKG_FORCE_SYSTEM_BINARIES=1
RUN mkdir -p "${VCPKG_ROOT}" "${VCPKG_DOWNLOADS}" \
	&& chown -R 1000:1000 "${VCPKG_ROOT}" "${VCPKG_DOWNLOADS}" \
	&& sudo -u hypnos git clone --single-branch https://github.com/microsoft/vcpkg "${VCPKG_ROOT}" \
	&& cd "${VCPKG_ROOT}" \
	&& ./bootstrap-vcpkg.sh -disableMetrics \
	&& ln -s /opt/vcpkg/vcpkg /usr/local/bin/vcpkg \
	&& vcpkg --version \
	&& rm -rf \
		"${VCPKG_ROOT}/buildtrees" \
		"${VCPKG_ROOT}/packages" \
		"${VCPKG_DOWNLOADS}"/*

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
	--mount=type=cache,target=/var/lib/apt,sharing=locked \
	apt update \
	&& apt-get install --no-install-recommends --yes \
		cmake \
		ninja-build


ENV VCPKG_BINARY_SOURCES="http,http://localhost:15151/{name}/{version}/{sha},readwrite"

USER hypnos

# Comment out the following lines if you don't want to use Zsh.
RUN wget https://github.com/robbyrussell/oh-my-zsh/raw/master/tools/install.sh -O - | zsh || true \
	&& git clone https://github.com/zsh-users/zsh-autosuggestions         ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-autosuggestions     \
	&& git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting \
	&& git clone https://github.com/zsh-users/zsh-completions             ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-completions         \
	&& git clone https://github.com/supercrabtree/k                       ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/k
ADD .zshrc /home/hypnos
ENV SHELL /bin/zsh
