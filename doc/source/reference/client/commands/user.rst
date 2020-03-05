Gestion des utilisateurs
========================

.. program:: waarp-gateway

.. option:: user

   Commande de gestion des utilisateurs. Doit être suivi d'une commande
   spécifiant l'action souhaitée.

   Un utilisateur permet de s'authentifier sur l'interface d'administration de
   la gateway.

   .. option:: add

      Commande de création d'utilisateur.

      .. option:: -u <USERNAME>, --username=<USERNAME>

         Spécifie le nom du nouvel utilisateur créé. Doit être unique.

      .. option:: -p <PASS>, --password=<PASS>

         Le mot de passe du nouvel utilisateur.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 user add -u toto -p password

   .. option:: list

      Commande de listing de multiples utilisateurs.

      .. option:: -l <LIMIT>, --limit=<LIMIT>

         Limite le nombre maximum d'utilisateurs autorisés dans la réponse. Fixé
         à 20 par défaut.

      .. option:: -o <OFFSET>, --offset=<OFFSET>

         Fixe le numéro du premier utilisateur renvoyé.

      .. option:: -s <SORT>, --sort=<SORT>

         Spécifie l'attribut selon lequel les utilisateurs seront triés. Les
         choix possibles sont: tri par nom d'utilisateur (`username`).

      .. option:: --descending, -d

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 user list -l 10 -o 5 -s username -d

   .. option:: get <USER_ID>

      Commande de consultation d'utilisateur. L'identifiant de l'utilisateur
      souhaité doit être spécifié en argument de programme.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 user get 1

   .. option:: update <USER_ID>

      Commande de modification d'un utilisateur existant. L'identifiant numérique
      de l'utilisateur doit être renseigné en argument de programme.

      .. option:: -u <USERNAME>, --username=<USERNAME>

         Change le nom de l'utilisateur. Doit être unique.

      .. option:: -p <PASS>, --password=<PASS>

         Change le mot de passe de l'utilisateur.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 user update 1 -u titi -p new_password

   .. option:: delete <USER_ID>

      Commande de suppression d'utilisateur. L'identifiant de l'utilisateur à
      supprimer doit être spécifié en argument de programme.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 user delete 1