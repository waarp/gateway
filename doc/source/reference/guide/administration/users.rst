Gestion des utilisateurs
========================

La commande de gestion des :term:`utilisateurs<utilisateur>` est ``user``. Cette
commande doit ensuite être suivie d'une action. La liste complète des actions est
disponible :doc:`ici <../../client/user/index>`.

Ajouter un utilisateur
----------------------

Pour créer un utilisateur, la commande est ``user add``. Les options de commande
suivantes doivent être fournies:

- ``-u``: le nom de l'utilisateur
- ``-p``: le mot de passe

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' user add -u 'toto' -p 'sésame'


Modifier un utilisateur
-----------------------

Pour modifier un utilisateur existant, la commande est ``user update``. Cette
commande doit être suivie du nom de l'utilisateur à modifier. Les options de
commandes sont identiques à la commande ``add``. Il est possible d'omettre une
ou plusieurs options pour faire une mise à jour partielle.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' user update 'toto' -p 'sésame2'


Consulter les utilisateurs
--------------------------

Pour lister les utilisateurs de la *gateway*, la commande est ``user list``. Les
options de commande permettent de filtrer les résultats selon divers critères,
pour plus de détails, voir la :doc:`documentation<../../client/user/list>` de
la commande ``list``.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' user list


Supprimer un utilisateur
------------------------

Pour supprimer un utilisateur, la commande est ``user delete``, suivie ensuite du
nom de l'utilisateur à supprimer.

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' user delete 'toto'