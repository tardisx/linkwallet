#!/usr/bin/perl

open my $fh, "<", "version/version.go" || die "oops";
while (my $l =  <$fh>) {
    if ($l =~ m/const Tag = "(.+)"/) {
        $tag = $1;
        system ('git', 'tag', '-a', $tag, '-m', "version $tag for release") ;
        die "could not tag?\n" if $? != 0;
        system ('git', 'push', 'origin', $tag);
        die "could not push tag?\n" if $? != 0;
        exit 0;
    }
}

die "no version in version/version.go?\n";
