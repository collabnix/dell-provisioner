FROM centos AS builder
RUN  yum -y install epel-release && \
     yum -y install xorriso && \
     curl -o /dell.iso https://downloads.dell.com/FOLDER04066625M/1/DELL_MDSS_Consolidated_RDVD_6_5_0_1.iso && \
     osirrox -indev /dell.iso -extract /linux/mdsm/SMIA-LINUXX64.bin /dell.bin && \
     rm -f /dell.iso

FROM centos AS installed
COPY --from=builder /dell.bin /
COPY installer.properties /
RUN  yum -y install unzip && \
     /dell.bin -f /installer.properties

FROM centos
RUN  mkdir -p /opt/smcli/mdstoragemanager/client && \
     mkdir -p /var/opt/SM && \
     echo "BASEDIR=/opt/smcli/mdstoragemanager" > /var/opt/SM/LAUNCHER_ENV
COPY --from=installed /opt/dell/mdstoragemanager/jre /opt/smcli/mdstoragemanager/jre
COPY --from=installed /opt/dell/mdstoragemanager/client/SMcli /opt/smcli/mdstoragemanager/client/
COPY --from=installed /opt/dell/mdstoragemanager/client/*.jar /opt/smcli/mdstoragemanager/client/

ENTRYPOINT ["/opt/smcli/mdstoragemanager/client/SMcli"]
