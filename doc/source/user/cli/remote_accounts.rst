Gestion des comptes distants
============================

La commande de gestion des :term:`comptes distants<compte distant>` est ``account
remote``. Cette commande doit ensuite être suivie du nom du partenaire auquel
sont rattachés le ou les comptes traités.

La commande doit donc ressembler donc à ceci:

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' account remote 'PARTENAIRE' *action*


La liste complète des actions est disponible :any:`ici
<reference-cli-client-remote-accounts>`.


Ajouter un compte distant
-------------------------

Pour ajouter un compte sur un partenaire, la commande est ``account remote add``.
Les options de commande suivantes doivent être fournies:

- ``-l``: le login du compte
- ``-p``: le mot de passe du compte

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' account remote 'WAARP R66' add -l 'titi' -p 'sésame'


Modifier un compte distant
--------------------------

Pour modifier un compte existant, la commande est ``account remote update``.
Cette commande doit être suivie du nom du compte à modifier. Les options de
commandes sont identiques à la commande ``add``. Il est possible d'omettre une
ou plusieurs options pour faire une mise à jour partielle.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' account remote 'WAARP R66' update 'titi' -p 'sésame2'


Consulter les comptes distants
------------------------------

Pour lister les comptes d'un partenaire, la commande est ``account remote list``.
Les options de commande permettent de filtrer les résultats selon divers critères,
pour plus de détails, voir la :any:`documentation
<reference-cli-client-remote-accounts-list>`
de la commande ``list``.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' account remote 'WAARP R66' list

Pour consulter un compte en particulier, la commande est ``get`` suivie du nom
de compte.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' account remote 'WAARP R66' get 'titi'


Supprimer un compte distant
---------------------------

Pour supprimer un compte, la commande est ``account remote delete``, suivi
ensuite du nom du compte à supprimer.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' account remote 'WAARP R66' delete 'titi'
