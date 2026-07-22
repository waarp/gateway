#########
Migration
#########

Apres avoir réaliser une montée de version du package de waarp-gateway,
une mise a jour du schéma de la base de données est nécessaire.

Arréter le service
==================

Tout d'abord arréter le service waarp-gatewayd

Package (deb/rpm)

.. code-block:: shell
   systemctl stop waarp-gatewayd

Archive autoportée (tar.gz/zip)

.. code-block:: shell
  ./bin/manage.sh stop

Sauvegarder la base de données
==============================

Avant d'effectuer la migration il est nécessaire de faire une sauvegarde de la base de données.

MySQL 
-----

Package (deb/rpm)

.. code-block:: shell
  cp /var/lib/waarp-gateway/db/waarp-gateway.db /var/lib/waarp-gateway/db/waarp-gateway.db.backup

Archive autoportée (tar.gz/zip)

.. code-block:: shell
  cp ./data/db/waarp-gateway.db ./data/db/waarp-gateway.db.backup

PostgreSQL
----------

.. code-block:: shell
  pg_dump waarp_gateway > waarp-gateway.sql.backup


Réaliser la migration du modèle de données
==========================================

Package (deb/rpm)

.. code-block:: shell
   waarp-gatewayd migrate -c /etc/waarp-gateway/gatewayd.ini

Archive autoportée (tar.gz/zip)

.. code-block:: shell
   ./bin/waarp-gatewayd migrate -c ./etc/gatewayd.ini

Redémarer le service
====================

Package (deb/rpm)

.. code-block:: shell
   systemctl start waarp-gatewayd

Archive autoportée (tar.gz/zip)

.. code-block:: shell
  ./bin/manage.sh start

