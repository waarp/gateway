####################
Gestion des serveurs
####################

La commande de gestion des :term:`serveurs<serveur>` locaux est ``server``.
Cette commande doit ensuite être suivie d'une action. La liste complète des
actions est disponible :any:`ici <reference-cli-client-servers>`.

.. note::
   Toutes modification ou ajout d'un serveur local ne prendra effet qu'après
   redémarrage de la *gateway*.


Ajouter un serveur
==================

Pour créer un serveur, la commande est ``server add``. Les options de commande
suivantes doivent être fournies:

- ``-n``: le nom du serveur
- ``-p``: le protocole de transfert
- ``-r``: la racine du serveur
- ``-c``: la :any:`configuration protocolaire <reference-proto-config>` du
  serveur

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server add -n 'WAARP SFTP' -p 'sftp' -r '/waarp_gw/sftp' -c '{"address":"localhost","port":8022}'


Modifier un serveur
===================

Pour modifier un serveur existant, la commande est ``server update``. Cette
commande doit être suivie du nom du serveur à modifier. Les options de commandes
sont identiques à la commande ``add``. Il est possible d'omettre une ou plusieurs
options pour faire une mise à jour partielle.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server update 'WAARP SFTP' -c '{"address":"localhost","port":8023}'


Consulter les serveurs
======================

Pour lister les serveurs de la *gateway*, la commande est ``server list``. Les
options de commande permettent de filtrer les résultats selon divers critères,
pour plus de détails, voir la :any:`référence <reference-cli-client-servers-list>` de
la commande ``list``.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server list

Pour consulter un serveur en particulier, la commande est ``get`` suivie du nom
du serveur.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server get 'WAARP SFTP'


Supprimer un serveur
====================

Pour supprimer un serveur, la commande est ``server delete``, suivie ensuite du
nom du serveur à supprimer.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server delete 'WAARP SFTP'
