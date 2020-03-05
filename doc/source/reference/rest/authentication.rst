Authentification
################

L'authentification des requêtes REST se fait au moyen de l'authentification
`HTTP basique <https://tools.ietf.org/html/rfc7617>`_.

Ce schéma d'authentification ce fait au moyen du header HTTP *Authorization*.
Pour s'authentifier, le client doit :

1. Obtenir le login et le mot de passe de l'utilisateur
2. Construire l'identifiant de l'utilisateur en concaténant le login,
   un caractère deux-points (":") et le mot de passe
3. Encoder l'identifiant obtenu en Base64
4. Préfixer l'identifiant encodé par une déclaration du schéma d'authentification
   basique ("Basic")


Par exemple, si l'utilisateur entre comme nom "Aladdin" et comme mot de passe
"open sesame", alors le client doit envoyer le header suivant :

    *Authorization:* Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==



En cas d'échec de l'authentification d'une requête, le serveur répondra par
un code HTTP ``401 - Unauthorized``.

Par défaut, la gateway ne possède qu'un seul utilisateur "admin" (mot de passe:
"admin_password") avec tous les droits,afin de permettre la mise en place la
configuration initiale de la gateway. Pour des raisons de sécurité, il est
fortemment recommandé lors de l'installation de la gateway de créer de nouveaux
utilisateurs avec des droits plus restreints, puis de supprimer cet utilisateur
"admin".