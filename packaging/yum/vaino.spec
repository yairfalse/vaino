Name:           vaino
Version:        %{version}
Release:        1%{?dist}
Summary:        Git diff for infrastructure - simple drift detection

License:        MIT
URL:            https://github.com/yairfalse/vaino
Source0:        https://github.com/yairfalse/vaino/releases/download/v%{version}/vaino_Linux_x86_64.tar.gz

BuildRequires:  systemd
Requires:       git
Recommends:     terraform

%description
VAINO (What's Going On) is a comprehensive infrastructure drift detection tool
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
install -D -m 755 vaino %{buildroot}%{_bindir}/vaino

# Install completions
install -D -m 644 completions/vaino.bash %{buildroot}%{_datadir}/bash-completion/completions/vaino
install -D -m 644 completions/vaino.zsh %{buildroot}%{_datadir}/zsh/site-functions/_vaino
install -D -m 644 completions/vaino.fish %{buildroot}%{_datadir}/fish/vendor_completions.d/vaino.fish

# Install documentation
install -D -m 644 README.md %{buildroot}%{_docdir}/%{name}/README.md
install -D -m 644 LICENSE %{buildroot}%{_docdir}/%{name}/LICENSE

%files
%{_bindir}/vaino
%{_datadir}/bash-completion/completions/vaino
%{_datadir}/zsh/site-functions/_vaino
%{_datadir}/fish/vendor_completions.d/vaino.fish
%doc %{_docdir}/%{name}/README.md
%license %{_docdir}/%{name}/LICENSE

%post
echo "VAINO installed successfully!"
echo "Run 'vaino version' to verify installation"
echo "Run 'vaino --help' to get started"

%changelog
* Thu Jan 01 2024 Yair <yair@example.com> - %{version}-1
- Initial package release