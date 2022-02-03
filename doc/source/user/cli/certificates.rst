#######################
Gestion des certificats
#######################

Il est possible d'attacher un :term:`certificat` à un agent de la gateway. Cet
agent peut être un :term:`serveur`, un :term:`partenaire`, un :term:`compte local`
ou un :term:`compte distant`.

Les commandes de gestion des certificats sont donc :

- ``server cert 'SERVEUR'`` pour gérer les certificats d'un serveur
- ``partner cert 'PARTENAIRE'`` pour gérer les certificats d'un partenaire
- ``account local cert 'LOGIN'`` pour gérer les certificats d'un compte local
- ``account remote cert 'LOGIN'`` pour gérer les certificats d'un compte distant

Ces commandes doivent être suivies de l'action souhaitée.


Ajouter un certificat
=====================

Pour ajouter un certificat à un agent, l'action est ``add``. Les options de
commande suivantes doivent être fournies:

- ``-n``: le nom du certificat
- ``-c``: le fichier contenant le certificat, encodé en format PEM
- ``-p``: le fichier contenant la clé privée du certificat
- ``-b``: le fichier contenant la clé publique du certificat

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server cert 'WAARP SFTP' add -n 'Clé SFTP' -b 'sftp.pub'


Modifier un certificat
======================

Pour modifier un certificat existant, l'action est ``update``. La commande doit
être suivie du nom du partenaire à modifier. Les options de commandes sont
identiques à l'action ``add``. Il est possible d'omettre une ou plusieurs
options pour faire une mise à jour partielle.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server cert 'WAARP SFTP' update 'Clé SFTP' -b 'new_sftp.pub'


Consulter les certificats
=========================

Pour lister les certificats d'un agent, l'action est ``list``. Les options de
commande permettent de filtrer les résultats selon divers critères, pour plus
de détails, voir la :any:`documentation
<reference-cli-client-servers-certs-list>` de la commande ``list``.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server cert 'WAARP SFTP' list

Pour consulter un certificat en particulier, la commande est ``get`` suivie du
nom du certificat.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server cert 'WAARP SFTP' get 'Clé SFTP'


Supprimer un certificat
=======================

Pour supprimer un certificat, l'action est ``delete``, suivie ensuite du nom du
certificat à supprimer.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server cert 'WAARP SFTP' delete 'Clé SFTP'
