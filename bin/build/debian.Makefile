#!/usr/bin/make -f
%:
	dh $@

override_dh_strip:
	dh_strip --exclude gcy
