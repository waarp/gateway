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

Vous pouvez également télécharger la dernière version du fichier DEB sur notre
`page de téléchargements`_.

Installez le DEB avec la commande :

.. code-block:: bash

   rpm -i waarp-gateway-[version].rpm


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
