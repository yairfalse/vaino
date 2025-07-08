Name:           wgo
Version:        %{version}
Release:        1%{?dist}
Summary:        Git diff for infrastructure - simple drift detection

License:        MIT
URL:            https://github.com/yairfalse/wgo
Source0:        https://github.com/yairfalse/wgo/releases/download/v%{version}/wgo_Linux_x86_64.tar.gz

BuildRequires:  systemd
Requires:       git
Recommends:     terraform

%description
WGO (What's Going On) is a comprehensive infrastructure drift detection tool
that helps you track changes in your infrastructure over time.

Features:
- Multi-provider support (Terraform, AWS, GCP, Kubernetes)
- Smart auto-discovery
- Time-based comparisons
- Unix-style output
- Zero configuration

%prep
%setup -q -c

%install
install -D -m 755 wgo %{buildroot}%{_bindir}/wgo

# Install completions
install -D -m 644 completions/wgo.bash %{buildroot}%{_datadir}/bash-completion/completions/wgo
install -D -m 644 completions/wgo.zsh %{buildroot}%{_datadir}/zsh/site-functions/_wgo
install -D -m 644 completions/wgo.fish %{buildroot}%{_datadir}/fish/vendor_completions.d/wgo.fish

# Install documentation
install -D -m 644 README.md %{buildroot}%{_docdir}/%{name}/README.md
install -D -m 644 LICENSE %{buildroot}%{_docdir}/%{name}/LICENSE

%files
%{_bindir}/wgo
%{_datadir}/bash-completion/completions/wgo
%{_datadir}/zsh/site-functions/_wgo
%{_datadir}/fish/vendor_completions.d/wgo.fish
%doc %{_docdir}/%{name}/README.md
%license %{_docdir}/%{name}/LICENSE

%post
echo "WGO installed successfully!"
echo "Run 'wgo version' to verify installation"
echo "Run 'wgo --help' to get started"

%changelog
* Thu Jan 01 2024 Yair <yair@example.com> - %{version}-1
- Initial package release