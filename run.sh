#!/bin/bash
if [ -f webdavz.log ]
	 then
	 cp webdavz.log webdavz.log.`date +%F`
fi
./webdavz >webdavz.log &
