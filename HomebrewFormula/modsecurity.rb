# Documentation: https://docs.brew.sh/Formula-Cookbook
#                https://rubydoc.brew.sh/Formula
# PLEASE REMOVE ALL GENERATED COMMENTS BEFORE SUBMITTING YOUR PULL REQUEST!
class Modsecurity < Formula
  desc "ModSecurity is an open source, cross platform web application firewall (WAF) engine for Apache, IIS and Nginx that is developed by Trustwave's SpiderLabs. It has a robust event-based programming language which provides protection from a range of attacks against web applications and allows for HTTP traffic monitoring, logging and real-time analysis. With over 10,000 deployments world-wide, ModSecurity is the most widely deployed WAF in existence. "
  homepage "https://www.modsecurity.org"
  url "https://github.com/SpiderLabs/ModSecurity/releases/download/v3.0.7/modsecurity-v3.0.7.tar.gz"
  sha256 "cfd8b7e7e6a0e9ca4e19b9adeb07594ba75eba16a66da5e9b0974c0117c21a34"
  license "Apache-2.0"

  depends_on "autoconf" => :build
  depends_on "automake" => :build
  depends_on "bison" => :build
  depends_on "doxygen" => :build
  depends_on "flex" => :build
  depends_on "libtool" => :build
  depends_on "pkg-config" => :build

  depends_on "curl"
  depends_on "geoip"
  depends_on "pcre"
  depends_on "pcre2"
  depends_on "libffi"
  depends_on "libxml2"
  depends_on "luarocks"
  depends_on "ssdeep"
  depends_on "yajl"
  depends_on "zlib"


  def install
    # ENV.deparallelize  # if your formula fails when building in parallel
    # Remove unrecognized options if warned by configure
    # https://rubydoc.brew.sh/Formula.html#std_configure_args-instance_method

    args = %W[
      --disable-debug
      --disable-silent-rules
      --prefix=#{prefix}
      --with-yajl=#{Formula["yajl"].opt_prefix}
      --with-geoip=#{Formula["geoip"].opt_prefix}
      --with-ssdeep=#{Formula["ssdeep"].opt_prefix}
      --with-lua=#{Formula["luarocks"].opt_prefix}
      --with-curl=#{Formula["curl"].opt_prefix}
      --with-libxml=#{Formula["libxml2"].opt_prefix}
      --with-pcre=#{Formula["pcre"].opt_prefix}
      --with-pcre2=#{Formula["pcre2"].opt_prefix}
    ]
    
    system "./build.sh"
    system "./configure", *args
    system "make", "install"
  end

  test do
    # `test do` will create, run in and delete a temporary directory.
    #
    # This test will fail and we won't accept that! For Homebrew/homebrew-core
    # this will need to be a test that verifies the functionality of the
    # software. Run the test with `brew test ModSecurity`. Options passed
    # to `brew install` such as `--HEAD` also need to be provided to `brew test`.
    #
    # The installed folder is not in the path, so use the entire path to any
    # executables being tested: `system "#{bin}/program", "do", "something"`.
    system "false"
  end
end
