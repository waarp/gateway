##########################
Gestion des comptes locaux
##########################

La commande de gestion des :term:`comptes locaux<compte local>` est ``account
local``. Cette commande doit ensuite être suivie du nom du serveur local auquel
sont rattachés le ou les comptes traités.

La commande doit donc ressembler donc à ceci:

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' account local 'SERVEUR' *action*


La liste complète des actions est disponible :any:`ici
<reference-cli-client-local-accounts>`.


Ajouter un compte local
=======================

Pour ajouter un compte sur un serveur, la commande est ``account local add``.
Les options de commande suivantes doivent être fournies:

- ``-l``: le login du compte
- ``-p``: le mot de passe du compte

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' account local 'WAARP SFTP' add -l 'toto' -p 'sésame'


Modifier un compte local
========================

Pour modifier un compte existant, la commande est ``account local update``. Cette commande
doit être suivie du nom du compte à modifier. Les options de commandes sont
identiques à la commande ``add``. Il est possible d'omettre une ou plusieurs
options pour faire une mise à jour partielle.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' account local 'WAARP SFTP' update 'toto' -p 'sésame2'


Consulter les comptes locaux
============================

Pour lister les comptes d'un serveur, la commande est ``list``. Les options
de commande permettent de filtrer les résultats selon divers critères, pour plus
de détails, voir la :any:`documentation
<reference-cli-client-local-accounts-list>` de la commande ``list``.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' account local 'WAARP SFTP' list

Pour consulter un compte en particulier, la commande est ``get`` suivie du nom
de compte.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' account local 'WAARP SFTP' get 'toto'


Supprimer un compte local
=========================

Pour supprimer un compte, la commande est ``account local delete``, suivie ensuite du nom du
compte à supprimer.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' account local 'WAARP SFTP' delete 'toto'
