.. _service_management:

##################
Gestion du service
##################


Avec systemd
============

Sur les systèmes fonctionnant avec ``systemd``, dont RHEL 7, les
commandes usuelles de gestions des services peuvent être utilisées.

L'unité ``systemd`` pour Waarp Gateway est fournie dans les packages sur les
systèmes utilisant systemd. Si vous utilisez les archives autonomes,
voir ci-dessous la :ref:`procédure d'ajout du service <systemd-service-unit>`.


.. code-block:: bash

   # Démarrage du service
   systemctl start waarp-gatewayd

   # Arrêt du service
   systemctl stop waarp-gatewayd

   # Statut du service
   systemctl status waarp-gatewayd

   # Redémarrage du service
   systemctl restart waarp-gatewayd

   # Activation du service au démarrage du serveur
   systemctl enable waarp-gatewayd

   # Désactivation du service au démarrage du serveur
   systemctl disable waarp-gatewayd



.. _systemd-service-unit:

Utilisation de systemd avec les archives autonomes
--------------------------------------------------

Si votre système d'exploitation utilise systemd, vous pouvez gérer
votre instance de Waarp Gateway avec.

Créez le fichier :file:`/etc/systemd/system/waarp-gateway.service` avec le
contenu suivant, en remplaçant :file:`/path/to/archive/root` par le chemin
vers le dossier d'extraction de l'archive :

.. code-block:: ini

   [Unit]
   Description=Waarp Gateway server

   [Service]
   Type=simple
   WorkingDirectory=/path/to/archive/root
   ExecStart=/bin/sh -c 'PATH=./share/:./bin/:$PATH exec ./bin/waarp-gatewayd server -c ./etc/gatewayd.ini'
   Restart=on-failure

   [Install]
   WantedBy=multi-user.target

Pour activer le démarrage automatique de Waarp Gateway au démarrage du
serveur, utilisez la commande :

.. code-block:: bash

   systemctl enable waarp-gatewayd


Avec SysVinit
=============

Sur les systèmes fonctionnant avec ``SysVinit``, dont RHEL 6, les
commandes usuelles de gestions des services peuvent être utilisées.

Le script d'init pour Waarp Gateway est fourni dans les packages sur les
systèmes utilisant SysVinit.


.. code-block:: bash

   # Démarrage du service
   /etc/init.d/waarp-gatewayd start

   # Arrêt du service
   /etc/init.d/waarp-gatewayd stop

   # Statut du service
   /etc/init.d/waarp-gatewayd status

   # Redémarrage du service
   /etc/init.d/waarp-gatewayd restart

   # Activation du service au démarrage du serveur
   update-rc.d waarp-gatewayd defaults # Systèmes basés sur Debian
   chkconfig --add waarp-gatewayd      # Systèmes basés sur Red Hat

   # Désactivation du service au démarrage du serveur
   update-rc.d -f waarp-gatewayd remove # Systèmes basés sur Debian
   chkconfig --del waarp-gatewayd       # Systèmes basés sur Red Hat



Avec les archives autonomes
===========================

Linux
-----

Le service se gère avec le script ``manage.sh`` situé dans le dossier
``bin`` à la racine du dossier d'extraction de l'archive :

.. code-block:: bash

  ./bin/manage.sh <commande>

Les commandes suivantes sont disponibles :

Commande ``manage.sh start``
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Démarre Waarp Gateway.

.. note::

  Si le port choisi est inférieur à 1024, le service doit être lancé
  avec l'utilisateur root.

Le nombre de CPU utilisés par Waarp Gateway peut être défini par la
variable d’environnement :envvar:`GOMAXPROCS`. Par défaut, le nombre de cœurs
CPU du serveur est utilisé.

Codes de retour :

===== =============
Code  Signification
===== =============
``0`` Le lancement de l'application a réussi
``1`` Le lancement a échoué. La raison de l'échec peut se trouver un des :ref:`fichiers de traces <log-management>`.
``2`` Le serveur est déjà lancé
===== =============



Commande ``manage.sh stop``
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Lance la procédure d'arrêt de Waarp Gateway. Le script attend 2 minutes
que Waarp Gateway s'arrête. Passé ce délais, le script rend la main,
**mais la procédure d'arrêt continue. L'arrêt définitif de Waarp Gateway
interviendra dès que tous les processus internes en cours seront terminés**.
Les codes retours suivants sont possibles :

Codes de retour :

===== =============
Code  Signification
===== =============
``0`` L'arrêt de l'application a réussi
``1`` L'arrêt a échoué. La raison de l'échec peut se trouver un des :ref:`fichiers de traces <log-management>`.
``2`` Le serveur est déjà arrêté
``3`` L'arrêt est en cours, mais la procédure d'arrêt n'est pas encore terminée.
===== =============



Commande ``manage.sh restart``
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Redémarre Waarp Gateway.

===== =================================
Code  Signification
===== =================================
``0`` Le redémarrage de l'application a réussi
``1`` Le redémarrage a échoué. La raison de l'échec peut se trouver un des :ref:`fichiers de traces <log-management>`.
``2`` Le serveur est déjà arrêté
``3`` L'arrêt est en cours, mais la procédure d'arrêt n'est pas encore terminée.
===== =================================



Commande ``manage.sh force-stop``
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Envoi un signal KILL à Waarp Gateway

Codes de retour :

===== =============
Code  Signification
===== =============
``0`` L'application est démarrée.
``1`` L'application est arrêtée.
``2`` Le fichier contenant l'identifiant du processus n'a pas été trouvé ou ne peut pas être lu. Le statut est inconnu
===== =============




Commande ``manage.sh status``
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Retourne l'état démarré/arrêté du serveur. Les codes retours suivants
sont possibles :

Codes de retour :

===== =============
Code  Signification
===== =============
``0`` L'application est démarrée.
``1`` L'application est arrêtée.
``2`` Le fichier contenant l'identifiant du processus n'a pas été trouvé ou ne peut pas être lu. Le statut est inconnu
===== =============


Windows
-------

Aucune gestion du service n'est actuellement fourni pour Windows.
