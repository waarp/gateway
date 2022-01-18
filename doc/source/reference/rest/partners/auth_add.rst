Ajouter une valeur d'authentification
=====================================

.. http:post:: /api/partners/(string:partner_name)/authentication

   Ajoute une nouvelle valeur d'authentification pour le partenaire donné.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom de la valeur d'authentification. Par défaut, le
     nom du type est utilisé.
   :reqjson string type: Le type d'authentification utilisé. Voir les
     :ref:`reference-auth-methods` pour la liste des différents type d'authentification
     supportés.
   :reqjson string value: La valeur primaire d'authentification.
   :reqjson string value2: La valeur secondaire d'authentification (dépend du
     type d'authentification).

   :statuscode 201: La valeur d'authentification a été créée avec succès.
   :statuscode 400: Un ou plusieurs des paramètres du serveur sont invalides.
   :statuscode 401: Authentification d'utilisateur invalide.

   :resheader Location: Le chemin d'accès au nouveau serveur créé.


   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/partners/openssh HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 2410

         {
           "name": "openssh_hostkey",
           "type": "ssh_public_key",
           "value": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDhbxVecyg3NbOuGgIbzUuB3GyVIKBRWhYUOaEJtqMR8ckb3WM6cy0yplbZ6is4y8gqGhE9pQ8g3JrbYYlrb8/HnjuCnSzA9BVhMNUxp/9Ar7GSvBO2bPIcPYBePe19AJ6MsjoT2jcZhUwlsiacHAnRWaOfYeJQP0Fw9zqhhPcjOnWIewNQaghwBXyyzQB/BbiYAMvPo0uveYY+Yr18ExIv3ybtqgAgSVHji4Jg4JFwVd9VPfAz3y4ucEYiOr/4bkOBTuAMxbvE+S8mvbOTQ+itsFQxuJgWTrx/53Yth3QYDwgjTaT7TLSSRpi1+s9QQg6XTanJyjtEmmYbnaB+EhAQfI0mfOripP/1cTq9StZfYTKl58ObrYWmc5CDH338uCdK5GxIP9eNz4RcLqPLvcVBrm62qsYReoD62InykggeOSgkOo4UGbC7JSEdW3afMBGdh797eht6qX3ywKbs7GNVwOt2M7xrpmCehU1uegN7GtIRvCZR0JH4+KSGitWFY3E=",
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/partners/openssh/authentication/openssh_hostkey
