Consulter un utilisateur
========================

.. http:get:: /api/users/(string:username)

   Renvoie l'utilisateur demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: L'utilisateur a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: L'utilisateur demandé n'existe pas

   :resjson string username: Le nom de l'utilisateur
   :resjson object perms: Les droits de l'utilisateur. Chaque attribut correspond
      à un élément sur lequel l'utilisateurs peut agir, et leur valeur indique
      les actions autorisées. Les différentes actions possibles sont lecture (``r``),
      écriture (``w``) et suppression (``d``). Ces droits sont renseignés avec une
      syntaxe similaire à `chmod <https://fr.wikipedia.org/wiki/Chmod#Modes>`_ où
      l'autorisation d'exécution a été remplacée par la suppression.

      * ``transfers`` (*string*) - Les droits sur les transferts. (*Note*:
         les transferts ne peuvent pas être supprimés).
      * ``servers`` (*string*) - Les droits sur les serveurs locaux.
      * ``partners`` (*string*) - Les droits sur les partenaires distants.
      * ``rules`` (*string*) - Les droits sur les règles de transfert.
      * ``users`` (*string*) - Les droits sur les autres utilisateurs.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/users/toto HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 105

      {
        "username": "toto",
        "perms": {
          "transfers":"rw-",
          "servers":"r--",
          "partners":"r--",
          "rules":"rwd",
          "users":"---"
        }
      }
