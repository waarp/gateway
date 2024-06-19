#######################
Gestion des certificats
#######################

Il est possible d'attacher des :term:`identifiants <information d'authentification>`
à un agent de la gateway. Cet agent peut être un :term:`serveur`,
un :term:`partenaire`, un :term:`compte local` ou un :term:`compte distant`.

Les commandes de gestion des identifiants sont donc :

- ``server credential 'SERVEUR'`` pour gérer les identifiants d'un serveur
- ``partner credential 'PARTENAIRE'`` pour gérer les identifiants d'un partenaire
- ``account credential cert 'LOGIN'`` pour gérer les identifiants d'un compte local
- ``account credential cert 'LOGIN'`` pour gérer les identifiants d'un compte distant

Ces commandes doivent être suivies de l'action souhaitée.


Ajouter un identifiant
======================

Pour ajouter un identifiant à un agent, l'action est ``add``. Les options de
commande suivantes doivent être fournies:

- ``-n``: le nom du certificat
- ``-t``: le type d'identifiant
- ``-v``: la valeur d'authentification (certificat, mot de passe)
- ``-s``: la deuxième valeur d'authentification (si la méthode en requiert une)

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' server credential 'WAARP SFTP' add -n 'Clé SFTP' -t 'ssh_public_key' -v 'sftp.pub'


Consulter les identifiants
==========================

Les identifiants d'un agent sont listés dans les informations de l'agent
lui-même, à la section "Credentials". Pour les récupérer, il suffit donc de
consulter les informations de l'agent en question.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' server get 'WAARP SFTP'

Pour avoir plus de détails sur un identifiant en particulier, utiliser
la commande ``credential get`` suivie du nom de l'identifiant.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' server credential 'WAARP SFTP' get 'Clé SFTP'


Supprimer un identifiant
========================

Pour supprimer un identifiant, l'action est ``delete``, suivie ensuite du nom de
l'identifiant à supprimer.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' server credential 'WAARP SFTP' delete 'Clé SFTP'
