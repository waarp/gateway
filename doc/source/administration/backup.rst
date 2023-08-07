################################
Import/Export de base de données
################################

Il est possible d'importer et exporter directement le contenu de la base de
données de Waarp Gateway depuis/vers un fichier. Cela est utile lorsque plusieurs
instances de Waarp Gateway ont des éléments en communs (partenaires, règles...) afin
d'éviter de devoir renseigner ces éléments plusieurs fois. Cela permet également
faire une sauvegarde du contenu de la base.

Les fichiers d'import/export sont en format JSON. Leur documentation complète
est disponible :any:`ici <reference-backup-json>`.

Export
======

Pour exporter la configuration d'une Waarp Gateway existante, la commande est
``waarp-gatewayd export``. Cette commande va récupérer les éléments demandés
depuis la base de données de Waarp Gateway, puis va les écrire dans un fichier.

Les options suivantes sont requises pour que la commande puisse s'exécuter :

- ``-c``: le fichier de configuration de Waarp Gateway (contient les informations
  de connexion à la base de données).
- ``-f``: le nom du fichier vers lequel les données seront exportées (par défaut
  le fichier sera nommé ``waarp-gateway-export.json``).
- ``-t``: les éléments à exporter, l'option peut être répétée pour exporter plusieurs
  éléments. Les valeurs acceptées sont ``rules``, ``servers``, ``partners``
  ou ``all``.

Par exemple, la commande suivante exporte les serveurs et les partenaires de la
Waarp Gateway vers le fichier ``gateway_backup.json`` :

.. code-block:: shell

   waarp-gatewayd export -c 'waarp-gateway.ini' -f 'gateway_backup.json' -t 'servers' -t 'partners'


Import
======

Pour importer un fichier de sauvegarde, la commande est ``waarp-gatewayd import``.
Cette commande va récupérer les éléments demandés dans le fichier donné, puis va
les insérer dans la base de données de la Gateway.
La commande requiert les options suivantes :

- ``-c``: le fichier de configuration de Waarp Gateway (contient les informations
  de connexion à la base de données).
- ``-s``: le fichier source de l'import.
- ``-t``: les éléments à importer. L'option peut être répétée pour importer plusieurs
  éléments, les valeurs acceptées sont ``rules``, ``servers``, ``partners``
  ou ``all``.
- ``-r``: les tables de la base de données seront vidées avant d'importer les
  nouveaux éléments depuis le fichier (seules les tables concernées par l'import
  seront vidées). Pour les scripts, l'option ``force-reset-before-import`` est
  disponible afin d'outrepasser le message de confirmation.

Par exemple, la commande suivante purge les serveurs et les partenaires de la base,
puis importe les nouveaux serveurs et les partenaires depuis le fichier
``gateway_backup.json`` et les insère dans la base de données :

.. code-block:: shell

   waarp-gatewayd import -r -c 'waarp-gateway.ini' -s 'gateway_backup.json' -t 'servers' -t 'partners'


La commande ``import`` inclue également une option ``-d`` ou ``--dry-run``
permettant de simuler l'import du fichier, mais sans appliquer les changements.
Cela peut être utile pour tester si le fichier source est dans un format correct
sans risquer d'insérer des éléments invalides dans la base de données.
