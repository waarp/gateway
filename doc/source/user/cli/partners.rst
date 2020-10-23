#######################
Gestion des partenaires
#######################

La commande de gestion des :term:`partenaires<partenaire>` distants est ``partner``.
Cette commande doit ensuite être suivie d'une action. La liste complète des actions
est disponible :any:`ici <reference-cli-client-partners>`.


Ajouter un partenaire
=====================

Pour créer un partenaire, la commande est ``partner add``. Les options de commande
suivantes doivent être fournies:

- ``-n``: le nom du partenaire
- ``-p``: le protocole de transfert
- ``-c``: la :any:`configuration protocolaire <reference-proto-config>` du partenaire

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' partner add -n 'WAARP R66' -p 'r66' -c '{"address":"waarp.org","port":8066}'


Modifier un partenaire
======================

Pour modifier un partenaire existant, la commande est ``partner update``. Cette
commande doit être suivie du nom du partenaire à modifier. Les options de commandes
sont identiques à la commande ``add``. Il est possible d'omettre une ou plusieurs
options pour faire une mise à jour partielle.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' partner update 'WAARP R66' -c '{"address":"waarp.fr","port":8068}'


Consulter les partenaires
=========================

Pour lister les partenaires de la *gateway*, la commande est ``partner list``.
Les options de commande permettent de filtrer les résultats selon divers critères,
pour plus de détails, voir la :any:`documentation
<reference-cli-client-partners-list>` de
la commande ``list``.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' partner list

Pour consulter un partenaire en particulier, la commande est ``get`` suivie du nom
du partenaire.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' partner get 'WAARP R66'


Supprimer un partenaire
=======================

Pour supprimer un partenaire, la commande est ``partner delete``, suivie ensuite
du nom du partenaire à supprimer.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' partner delete 'WAARP R66'
