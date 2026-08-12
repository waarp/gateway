.. _backup:

##########################
Sauvegarde et restauration
##########################

Il existe deux façons de sauvegarder une Waarp Gateway, qui n'ont ni le même
périmètre ni les mêmes contraintes. Elles sont complémentaires plutôt que
concurrentes.

:ref:`Par export JSON <backup-json>`
   La commande ``waarp-gatewayd export`` écrit la configuration dans un fichier
   JSON, que ``waarp-gatewayd import`` relit.

:ref:`Par copie de la base et de la clef <backup-database>`
   Une sauvegarde du :abbr:`SGBD (Système de Gestion de Base de Données)`,
   accompagnée du fichier contenant la clef de chiffrement.

.. list-table::
   :header-rows: 1
   :widths: 30 35 35

   * -
     - Export JSON
     - Base de données et clef
   * - Mise en œuvre
     - Une commande, indépendante du SGBD
     - Dépend du SGBD utilisé
   * - Nombre de fichiers
     - Un seul
     - Au moins deux : la base et la clef
   * - Secrets
     - **En clair** dans le fichier
     - Chiffrés, illisibles sans la clef
   * - Historique des transferts
     - **Non inclus**
     - Inclus
   * - Changement de SGBD
     - Possible
     - Non
   * - Restauration partielle
     - Oui, par catégorie d'éléments
     - Non, tout ou rien

L'export est donc l'outil adapté à la sauvegarde de la configuration, à une
migration de serveur, ou à un changement de :abbr:`SGBD (Système de Gestion de
Base de Données)` — un passage de SQLite à PostgreSQL par exemple, où une copie
de base ne serait d'aucun secours. La copie de base est en revanche la seule
méthode qui préserve l'historique des transferts.

Le format des fichiers d'import/export est documenté :any:`ici
<reference-backup-json>`.

.. warning::

   Sur une installation par paquet, lancez ces commandes sous l'identité du
   compte ``waarp``, et non en ``root``. Exécutée en ``root`` avant le premier
   démarrage du service, une de ces commandes crée une clef
   :file:`/etc/waarp-gateway/passphrase.aes` appartenant à ``root``, que le
   service ne pourra ensuite plus lire — il refusera alors de démarrer. Le
   compte ``waarp`` n'ayant pas de shell de connexion, ``runuser -u waarp --``
   est la forme à utiliser.


.. _backup-json:

Méthode 1 : export et import JSON
=================================

Sauvegarde
----------

La commande ``waarp-gatewayd export`` récupère les éléments demandés depuis la
base de données et les écrit dans un fichier.

- ``-c`` : le fichier de configuration de Waarp Gateway, qui contient les
  informations de connexion à la base.
- ``-f`` : le fichier de destination. En son absence, l'export est écrit sur la
  sortie standard.
- ``-t`` : limite l'export à certaines catégories d'éléments. L'option peut être
  répétée. **Par défaut, tout est exporté.** La liste des valeurs acceptées est
  donnée par :ref:`la documentation de la commande
  <reference-cmd-waarp-gatewayd-export>`.

Par exemple, pour exporter la totalité de la configuration :

.. code-block:: shell

   runuser -u waarp -- waarp-gatewayd export -c '/etc/waarp-gateway/gatewayd.ini' -f 'gateway_backup.json'

Et pour n'exporter que les serveurs et les partenaires :

.. code-block:: shell

   runuser -u waarp -- waarp-gatewayd export -c '/etc/waarp-gateway/gatewayd.ini' -f 'gateway_backup.json' -t 'servers' -t 'partners'

.. danger::

   **Le fichier d'export contient les secrets en clair.** Les mots de passe des
   comptes distants, les clefs privées TLS et SSH, les identifiants *cloud* et
   SMTP, les passphrases SNMPv3 et les clefs de chiffrement PGP et AES sont
   stockés chiffrés en base, mais l'export les déchiffre pour les écrire dans le
   JSON.

   C'est ce qui rend cette sauvegarde autonome — elle ne dépend pas de la clef
   de chiffrement — mais cela impose de protéger le fichier en conséquence :
   droits restreints, stockage chiffré, et en aucun cas un dépôt Git ou un
   partage réseau en clair.

Restauration
------------

La commande ``waarp-gatewayd import`` relit un fichier d'export et insère son
contenu dans la base.

- ``-c`` : le fichier de configuration de Waarp Gateway.
- ``-s`` : le fichier source de l'import. En son absence, il est lu sur l'entrée
  standard.
- ``-t`` : limite l'import à certaines catégories. Mêmes valeurs que pour
  l'export, et par défaut tout est importé. Voir :ref:`la documentation de la
  commande <reference-cmd-waarp-gatewayd-import>`.
- ``-r`` : vide les tables concernées avant d'importer. Pour un usage en script,
  ``--force-reset-before-import`` fait de même sans demander de confirmation.
- ``-d`` : simule l'import sans appliquer aucun changement. Utile pour vérifier
  qu'un fichier source est valide avant de l'appliquer réellement.

Par exemple, pour remplacer les serveurs et les partenaires existants par ceux
du fichier de sauvegarde :

.. code-block:: shell

   runuser -u waarp -- waarp-gatewayd import -r -c '/etc/waarp-gateway/gatewayd.ini' -s 'gateway_backup.json' -t 'servers' -t 'partners'

Les secrets étant en clair dans le fichier, ils sont rechiffrés à l'import avec
la clef de l'instance qui les reçoit. Une sauvegarde reste donc exploitable même
si cette clef n'est pas celle qui a servi à produire l'export, ce qui est le cas
lors d'une migration de serveur.


.. _backup-database:

Méthode 2 : copie de la base et de la clef
==========================================

Sauvegarde
----------

Cette méthode consiste à sauvegarder la base de données par les moyens propres à
votre :abbr:`SGBD (Système de Gestion de Base de Données)` — ``pg_dump``,
``mysqldump``, ou une copie du fichier pour SQLite, le service étant arrêté afin
d'obtenir un état cohérent.

.. danger::

   **La base seule ne suffit pas.** Les secrets y sont chiffrés, et la clef qui
   permet de les lire n'est pas dans la base : elle vit dans le fichier désigné
   par le paramètre ``AESPassphrase`` de la configuration, soit
   :file:`/etc/waarp-gateway/passphrase.aes` sur une installation par paquet.

   **Cette clef fait partie intégrante de la sauvegarde** et doit être copiée en
   même temps que la base, puis conservée avec le même niveau de protection.

Restauration
------------

Restaurez la base par les moyens de votre SGBD, puis remettez en place le
fichier de clef **avant** de démarrer le service.

.. danger::

   Restaurer une base sans sa clef est une perte de données **définitive et
   silencieuse**.

   Si la clef est absente au démarrage, Gateway en génère automatiquement une
   nouvelle. Le service démarre alors sans la moindre erreur : les règles, les
   utilisateurs, les partenaires et l'historique sont tous intacts, et rien
   n'indique qu'un problème existe. La panne n'apparaît qu'au premier transfert,
   sous la forme d'une erreur de déchiffrement des identifiants.

   Restaurer la clef d'origine *après coup* ne répare pas la situation : tout ce
   qui aura été écrit sous la clef intermédiaire deviendra illisible à son tour,
   et plus aucune clef ne permettra alors de lire l'intégralité de la base.

   Les clefs privées PGP et AES utilisées par les tâches de chiffrement sont
   concernées au même titre que les mots de passe : leur perte peut rendre
   indéchiffrables des fichiers déjà traités.


Historique des transferts
=========================

L'historique n'est pas couvert par ``export`` et ``import``. Il n'est sauvegardé
que par la :ref:`copie de la base <backup-database>`.

Il existe cependant un mécanisme dédié, dont la finalité première est
l'archivage plutôt que la sauvegarde : l'option ``-e`` de :ref:`la commande purge
<reference-cmd-waarp-gatewayd-purge>` écrit dans un fichier JSON les entrées
qu'elle supprime, et :ref:`la commande restore-history
<reference-cmd-waarp-gatewayd-restore-history>` permet de les réinjecter.

.. note::

   ``purge -e`` est une opération **destructrice** : elle exporte ce qu'elle
   efface. Elle ne peut donc pas servir à sauvegarder un historique que l'on
   souhaite conserver en base.
