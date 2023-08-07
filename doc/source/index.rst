Documentation de Waarp Gateway
==============================

:Version: |version|
:Date:    |today|

Waarp Gateway est une passerelle de rupture de transferts bidirectionnelle et
multi-protocolaire.

Ses principales fonctionnalités sont :

- Rupture protocolaire
- Interconnexion protocolaire
- Protocoles supportés : :ref:`R66 <ref-proto-r66>`, :ref:`SFTP
  <ref-proto-r66>`, :ref:`HTTP(S) <ref-proto-http>`
- Reprise de transferts
- Possibilité de :ref:`programmer des transferts <user-add-transfer>`
- Définition de :ref:`chaînes de traitements <user-add-rule>` pré-transfert,
  post-transfert et en cas d'erreur
- Fonctionne en :ref:`grappe haute disponibilité <reference-conf-override>`
- :ref:`API REST <reference-rest-api>` de gestion et d'administration
- Client d'`administration en ligne de commande <ref-cli>`_ local ou à distance

Cette documentation contient plusieurs sections.

La première vous accompagne dans l'installation de l'application :

.. toctree::
   :caption: Guide d'installation
   :maxdepth: 1

   installation/requirements
   installation/install

.. toctree::
   :caption: Premiers pas
   :maxdepth: 1

   getting-started/intro
   getting-started/start
   getting-started/folders
   getting-started/sftp-receive
   getting-started/r66-send
   getting-started/rebound
   getting-started/cluster

Le guide utilisateur décrit le fonctionnement de Waarp Transfer :

.. toctree::
   :caption: Guide utilisateur
   :maxdepth: 1

   user/cli/index

La section suivante contient plusieurs page ayant trait à l'administration et à
l'exploitation de l'application :

.. toctree::
   :caption: Guide administrateur
   :maxdepth: 1

   administration/usage
   administration/service
   administration/backup
   administration/purge

La dernière section présente la référence des diverses commandes et paramètres
de l'application.

.. toctree::
   :caption: Référence
   :maxdepth: 1

   reference/cli/index
   reference/configuration
   reference/override
   reference/rest/index
   reference/container
   reference/protocols/index
   reference/proto_config/index
   reference/auth_methods
   reference/tasks/index
   reference/cloud/index
   reference/errorcodes
   reference/backup

.. toctree::
   :caption: Annexes
   :maxdepth: 1

   changelog
   glossary
   genindex

