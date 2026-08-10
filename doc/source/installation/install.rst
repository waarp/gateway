.. _install:

############
Installation
############

Installation sur RHEL, Fedora, et Centos
========================================

Pour faciliter l'installation et les mises à jour de Waarp Gateway, nous
fournissons des dépôts pour :abbr:`RHEL (Red Hat Enterprise
Linux)` 7+, distributions dérivées (Centos/Scientific Linux) et Fedora.

Pour ajouter les dépôts Waarp à votre système, suivez la procédure
indiquée sur notre `page de présentation des dépôts`_.

Après avoir suivi cette procédure, vous pouvez installer Waarp Gateway
avec la commande :

.. code-block:: bash

   yum install waarp-gateway


Vous pouvez également télécharger la dernière version du fichier RPM sur notre
`page de téléchargements`_.

Installez le RPM avec la commande :

.. code-block:: bash

   rpm -i waarp-gateway-[version].rpm


Installation sur Debian et dérivés
==================================

Vous pouvez télécharger la dernière version du fichier DEB sur notre
`page de téléchargements`_.

Installez le DEB avec la commande :

.. code-block:: bash

   apt install ./waarp-gateway_[version]-1_amd64.deb

La commande ``dpkg -i`` fonctionne également, le paquet ne dépendant que de
composants présents sur tout système Debian.


.. _install-layout:

.. note::

   Le service n'est ni activé ni démarré par l'installation, la base de données
   devant être migrée au préalable. Voir :ref:`service_management` et
   :doc:`../administration/migrate`.

.. _install-deprecated-paths:

Emplacements dépréciés
----------------------

Jusqu'à la version 0.16.0 incluse, ``get-remote`` et ``updateconf`` étaient
installés dans :file:`/usr/share/waarp-gateway/`. Ce n'était pas conforme aux
politiques Debian et RPM, qui réservent :file:`/usr/share` aux données
indépendantes de l'architecture.

Les anciens chemins restent fonctionnels **jusqu'à la version 1.0** : ils sont
occupés par un script de compatibilité qui relaie l'appel vers le nouvel
emplacement, en affichant un avertissement. Si vos règles de transfert ou vos
scripts d'intégration utilisent le chemin absolu, remplacez-le par le simple
nom du programme (``updateconf``, ``get-remote``) : cette forme est résolue
correctement aussi bien sur une installation par paquet que sur une archive
portable ou dans un container.


Utilisation du container avec Docker
====================================

Les images Gateway peuvent être lancées avec la commande

.. code-block:: shell

   docker run code.waarp.fr:5000/apps/gateway/gateway:latest

Le tag ``latest`` pointe toujours vers la dernière version publiée.

L'instance de Gateway lancée dans le container par variables d'environnement.
Par exemple, pour définir un identifiant d'instance, et rediriger le port 8080
du container vers le port 8000 de l'hôte,  utilisez la commande suivante :

.. code-block:: shell

   docker run -e WAARP_GATEWAY_NAME=ma-gateway -p 8000:8080 code.waarp.fr:5000/apps/gateway/gateway:latest


Toutes les variables d'environnement prises en compte sont listées sur :any:`la page
dédiée de la documentation <ref-config-container>`

.. liens:
.. _page de téléchargements: https://dl.waarp.org/dist/waarp-gateway/
.. _page de présentation des dépôts: https://dl.waarp.org/repos/
