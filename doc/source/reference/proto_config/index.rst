.. _reference-proto-config:

##########################
Configuration protocolaire
##########################

Chaque serveur et partenaire de transfert de Waarp Gateway gère un protocole de
transfert particulier. Par conséquent, chacun de ces agents doit être accompagné
d'une configuration spécifique à ce protocole.

Cette configuration est stockée sous la forme d'un objet JSON dont la structure
varie en fonction du protocole. Cette rubrique documente la structure de ces
objets JSON. Lors de la création d'un serveur ou d'un partenaire, l'objet JSON
de configuration approprié doit être fourni.

.. note::
   La structure de l'objet de configuration peut être varier entre un serveur et
   un partenaire pour un même protocole.

.. toctree::
   :maxdepth: 2

   r66
   sftp
   http
   ftp
   pesit
   webdav
